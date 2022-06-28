execute 'curl -s https://packagecloud.io/install/repositories/ookla/speedtest-cli/script.deb.sh | bash' do
  not_if 'test -f /etc/apt/sources.list.d/ookla_speedtest-cli.list'

  notifies :run, 'execute[apt update]', :immediately
end

package 'speedtest'
