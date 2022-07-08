#!/bin/bash

set -e
TIME=$(date "+%Y%m%d_%H%M%S")
SERVICE=${SERVICE:-webapp}
WORKDIR=/home/isucon/${SERVICE}
LOGDIR=/home/isucon/logs/${TIME}
BRANCH=${BRANCH:-$(git rev-parse --abbrev-ref HEAD)}

echo -e "!!!!!!!!! RELOAD START !!!!!!!!!\nSTART=$TIME\nCOMMIT=$(git rev-parse --short HEAD)" | notify_slack

# checkout current branch
git fetch origin
git reset --hard $BRANCH
git pull origin $BRANCH

echo ":: BUILD APP         ====>"
cd $WORKDIR/golang
make clean
make
cd -

echo
echo ":: BACKUP            ====>"
mkdir -p $LOGDIR
if [ -e /var/log/mysql/mysql-slow.log ]; then
	sudo cp /var/log/mysql/mysql-slow.log ${LOGDIR}/mysql-slow.log
fi

if [ -e /var/log/nginx/access_log ]; then
	sudo cp /var/log/nginx/access.log ${LOGDIR}/nginx/access.log
fi

echo
echo ":: CLEAR LOGS        ====>"
sudo truncate -s 0 -c /var/log/nginx/access.log
sudo bash -c ':>/var/log/mysql/slow.log'
if [ "$(pgrep mysql | wc -l)" ]; then
	sudo mysqladmin flush-logs
fi

echo
echo ":: COPY CONFIGS      ====>"
if [ -e etc/sysctl.conf ]; then
  cp etc/sysctl.conf /etc/sysctl.conf
fi
sysctl -p

# import configs
sudo chown -R root:root $WORKDIR/etc/nginx
sudo chown -R root:root $WORKDIR/etc/mysql
sudo cp -r $WORKDIR/etc/nginx/* /etc/nginx
sudo cp -r $WORKDIR/etc/mysql/* /etc/mysql
sudo chown -R isucon:isucon $WORKDIR/etc/nginx
sudo chown -R isucon:isucon $WORKDIR/etc/mysql

echo
echo ":: RESTART SERVICES  ====>"
sudo systemctl daemon-reload
sudo systemctl restart nginx
sudo systemctl restart mysql
sudo systemctl restart web-golang

echo
echo ":: BENCHMARK         ====>"
cd $HOME
$HOME/bin/benchmarker

echo
echo ":: ACCESS LOG        ====>"
sudo cat /var/log/nginx/access.log | \
	alp ltsv --sort avg -r > /tmp/nginx_access.log.${TIME}

echo ":: SLOW QUERY DIGEST ====>"
sudo pt-query-digest /var/log/mysql/slow.log > /tmp/mysql_slow.log.${TIME}
echo "log time: $TIME"

ENDTIME=$(date "+%Y%m%d_%H%M%S")
echo -e "!!!!!!!!! RELOAD END AT $ENDTIME !!!!!!!!!\nLOGTIME=$TIME\nCOMMIT=$(git rev-parse --short HEAD)" | notify_slack

