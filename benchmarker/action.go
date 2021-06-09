package main

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/isucon/isucandar"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucandar/worker"
)

func BrowserAccess(ctx context.Context, user *User, rpath string) error {
	req, err := user.Agent.GET(rpath)
	if err != nil {
		return failure.NewError(ErrCritical, err)
	}

	res, err := user.Agent.Do(ctx, req)
	if err != nil {
		return failure.NewError(ErrCritical, err)
	}

	if err := assertStatusCode(res, 200); err != nil {
		return err
	}

	resources, perr := user.Agent.ProcessHTML(ctx, res, res.Body)
	if perr != nil {
		return failure.NewError(ErrCritical, err)
	}

	for _, resource := range resources {
		if resource.Error != nil {
			var nerr net.Error
			if failure.As(resource.Error, &nerr) {
				if nerr.Timeout() || nerr.Temporary() {
					return failure.NewError(ErrTimeout, err)
				}
			}
			return failure.NewError(ErrInvalidAsset, fmt.Errorf("リソースの取得に失敗しました: %s: %v", resource.Request.URL.Path, resource.Error))
		}

		if resource.Response.StatusCode == 304 {
			continue
		}

		if err := assertStatusCode(resource.Response, 200); err != nil {
			return err
		}

		if err := assertChecksum(resource.Response); err != nil {
			return err
		}
	}

	return nil
}

type SignupResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Nickname  string    `json:"nickname"`
	CreatedAt time.Time `json:"created_at"`
}

func ActionSignup(ctx context.Context, step *isucandar.BenchmarkStep, u *User) error {
	if err := BrowserAccess(ctx, u, "/signup"); err != nil {
		return err
	}

	values := url.Values{}
	values.Add("email", u.Email)
	values.Add("nickname", u.Nickname)

	body := strings.NewReader(values.Encode())

	req, err := u.Agent.POST("/api/signup", body)
	if err != nil {
		// request が生成できないなんてのは相当やばい状況なのでたいてい Critical です
		// さっさと Critical エラーにして早めにベンチマーカー止めてあげるのも優しさ
		return failure.NewError(ErrCritical, err)
	}

	res, err := u.Agent.Do(ctx, req)
	if err != nil {
		return failure.NewError(ErrCritical, err)
	}

	hasError := false
	if err := assertStatusCode(res, 200); err != nil {
		step.AddError(err)
		hasError = true
	}

	if err := assertContentType(res, "application/json"); err != nil {
		step.AddError(err)
		hasError = true
	}

	jsonResp := &SignupResponse{}
	if err := assertJSONBody(res, jsonResp); err != nil {
		step.AddError(err)
		hasError = true
	} else {
		if err := assertEqualString(u.Email, jsonResp.Email); err != nil {
			step.AddError(err)
			hasError = true
		}

		if err := assertEqualString(u.Nickname, jsonResp.Nickname); err != nil {
			step.AddError(err)
			hasError = true
		}
	}

	if !hasError {
		u.ID = jsonResp.ID
		u.CreatedAt = jsonResp.CreatedAt
		step.AddScore(ScoreSignup)
	}

	return nil
}

// ユーザーをたくさんつくるよ
func ActionSignups(parent context.Context, step *isucandar.BenchmarkStep, s *Scenario) error {
	// とりあえず10秒くらい
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()

	wg := sync.WaitGroup{}
	wg.Add(1)

	// とりあえず50並列くらい
	w, err := worker.NewWorker(func(ctx context.Context, _ int) {
		select {
		case <-ctx.Done():
			// context が終わってたら抜ける
			// あ、Paralle だと一回しか実行しないのか
			return
		default:
		}

		wg.Add(1)
		defer wg.Done()

		user, err := s.NewUser()
		if err != nil {
			step.AddError(err)
			return
		}
		if err := ActionSignup(parent, step, user); err != nil {
			step.AddError(err)
			return
		}
		s.Users.Add(user)
	}, worker.WithMaxParallelism(s.Parallelism), worker.WithInfinityLoop())
	if err != nil {
		return err
	}

	// 一応ここでも待ち合わせはするんだけどね
	w.Process(ctx)

	// 確実に止める、止まったことを検知するために
	wg.Done()
	wg.Wait()

	return nil
}

type LoginResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Nickname  string    `json:"nickname"`
	CreatedAt time.Time `json:"created_at"`
}

// Action がエラーを返す → Action の失敗
// Action がエラーを返さない → Action としては成功。シナリオとしてはどうかわからない
func ActionLogin(ctx context.Context, step *isucandar.BenchmarkStep, u *User) error {
	if err := BrowserAccess(ctx, u, "/login"); err != nil {
		return err
	}

	values := url.Values{}

	if u.FailOnLogin {
		values.Add("email", "invalid-"+u.Email)
	} else {
		values.Add("email", u.Email)
	}

	body := strings.NewReader(values.Encode())

	req, err := u.Agent.POST("/api/login", body)
	if err != nil {
		// request が生成できないなんてのは相当やばい状況なのでたいてい Critical です
		// さっさと Critical エラーにして早めにベンチマーカー止めてあげるのも優しさ
		return failure.NewError(ErrCritical, err)
	}

	res, err := u.Agent.Do(ctx, req)
	if err != nil {
		return failure.NewError(ErrCritical, err)
	}

	hasError := false
	if u.FailOnLogin {
		if err := assertStatusCode(res, 403); err != nil {
			step.AddError(err)
			hasError = true
		}
	} else {
		if err := assertStatusCode(res, 200); err != nil {
			step.AddError(err)
			hasError = true
		}

		if err := assertContentType(res, "application/json"); err != nil {
			step.AddError(err)
			hasError = true
		}

		jsonResp := &LoginResponse{}
		if err := assertJSONBody(res, jsonResp); err != nil {
			step.AddError(err)
			hasError = true
		} else {
			if err := assertEqualString(u.Email, jsonResp.Email); err != nil {
				step.AddError(err)
				hasError = true
			}
			if err := assertEqualString(u.Nickname, jsonResp.Nickname); err != nil {
				step.AddError(err)
				hasError = true
			}
		}
	}

	if !hasError {
		step.AddScore(ScoreLogin)
	}

	return nil
}

func ActionLogins(parent context.Context, step *isucandar.BenchmarkStep, s *Scenario) error {
	usersCount := s.Users.Count()

	// とりあえず30秒耐える
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()

	wg := sync.WaitGroup{}
	wg.Add(1)

	// とりあえず100並列くらい
	w, err := worker.NewWorker(func(ctx context.Context, idx int) {
		select {
		case <-ctx.Done():
			// context が終わってたら抜ける
			return
		default:
		}

		user := s.Users.Get(idx)
		if err := ActionLogin(ctx, step, user); err != nil {
			step.AddError(err)
		}
	}, worker.WithMaxParallelism(s.Parallelism), worker.WithLoopCount(int32(usersCount)))
	if err != nil {
		return err
	}

	w.Process(ctx)

	wg.Done()
	wg.Wait()

	return nil
}

type CreateScheduleResponse struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Capacity  uint      `json:"capacity"`
	CreatedAt time.Time `json:"created_at"`
}

func ActionCreateSchedule(ctx context.Context, step *isucandar.BenchmarkStep, s *Scenario) (*Schedule, error) {
	user := s.StaffUser

	schedule, err := s.NewSchedule()
	if err != nil {
		return nil, err
	}

	values := url.Values{}
	values.Add("title", schedule.Title)
	values.Add("capacity", strconv.Itoa(int(schedule.Capacity)))

	body := strings.NewReader(values.Encode())
	req, err := user.Agent.POST("/api/schedules", body)
	if err != nil {
		return nil, failure.NewError(ErrCritical, err)
	}

	res, err := user.Agent.Do(ctx, req)
	if err != nil {
		return nil, failure.NewError(ErrCritical, err)
	}

	hasError := false
	// なんで return せずに AddError しているかというと
	// なるべく多くの検査項目をチェックしてあげて競技者にエラーを返さないと
	// ステータスコード直したら実は Content Type が狂ってた……みたいなバグ探しのベンチ試行回数が無駄に増えるので
	// なるべくたくさんチェックしてあげたいね、という意図です
	if err := assertStatusCode(res, 200); err != nil {
		step.AddError(err)
		hasError = true
	}

	if err := assertContentType(res, "application/json"); err != nil {
		step.AddError(err)
		hasError = true
	}

	jsonResp := &CreateScheduleResponse{}
	if err := assertJSONBody(res, jsonResp); err != nil {
		step.AddError(err)
		hasError = true
	} else {
		if err := assertEqualString(jsonResp.Title, schedule.Title); err != nil {
			step.AddError(err)
			hasError = true
		}
		if err := assertEqualUint(jsonResp.Capacity, schedule.Capacity); err != nil {
			step.AddError(err)
			hasError = true
		}
	}

	if !hasError {
		schedule.ID = jsonResp.ID
		schedule.CreatedAt = jsonResp.CreatedAt

		step.AddScore(ScoreCreateSchedule)
	}
	return schedule, nil
}

/*
	10個のスケジュールを作る
*/
func ActionCreateSchedules(ctx context.Context, step *isucandar.BenchmarkStep, s *Scenario) error {
	wg := sync.WaitGroup{}
	wg.Add(1)

	w, err := worker.NewWorker(func(ctx context.Context, _ int) {
		select {
		case <-ctx.Done():
			// context が終わってたら抜ける
			// あ、Paralle だと一回しか実行しないのか
			return
		default:
		}

		wg.Add(1)
		defer wg.Done()

		schedule, err := ActionCreateSchedule(ctx, step, s)
		if err != nil {
			step.AddError(err)
			return
		}
		s.Schedules.Add(schedule)
	}, worker.WithMaxParallelism(s.Parallelism), worker.WithLoopCount(10))
	if err != nil {
		return err
	}

	w.Process(ctx)

	wg.Done()
	wg.Wait()

	return nil
}

func ActionCreateReservation(ctx context.Context, step *isucandar.BenchmarkStep, schedule *Schedule, user *User) error {
	return BrowserAccess(ctx, user, "/")
}
