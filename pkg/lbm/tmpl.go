package lbm

var lvsTmpl = `
{{$prot := .Protocol}}
{{$vip := .VIP}}
{{$port := .Port}}
{{$option := .Conf}}
{{$monitor := .Monitor}}
{{$common := .KConf}}
{{$lname := .LAddrGroup}}
virtual_server_group {{$vip}}:{{$port}} {
	{{$vip}} {{$port}}
}

virtual_server group {{$vip}}:{{$port}} {
	delay_loop {{$monitor.Interval}}
	lb_algo {{if ne $option.Strategy ""}}{{$option.Strategy}}{{else}}rr{{end}}
	lb_kind {{if ne $option.Kind ""}}{{$option.Kind}}{{else}}FNAT{{end}}
	protocol {{if ne $prot "UDP"}}TCP{{else}}UDP{{end}}
	{{if $option.PersistenceTimeout}}persistence_timeout {{$option.PersistenceTimeout}}{{end}}
	{{if $option.SynProxy}}syn_proxy{{end}}
	{{if $option.BPSLimit}}bps_limit {{$option.BPSLimit}}{{end}}
	{{if $option.CPSLimit}}cps_limit {{$option.CPSLimit}}{{end}}

	{{if ne $lname ""}}laddr_group_name {{$lname}}{{end}}

	alpha
	omega
	quorum 1
	hysteresis 0
	quorum_up "ip addr add {{$vip}}/32 dev lo"
	quorum_down "ip addr del {{$vip}}/32 dev lo"

{{range $backend := .Servers}}
	real_server {{$backend.IP}} {{$backend.Port}} {
		weight {{$backend.Weight}}
		inhibit_on_failure
		{{if eq $monitor.Type "MISC"}}
		MISC_CHECK {
			misc_path "{{$common.MiscScript}} {{$backend.IP}} {{$backend.Port}}"
			misc_timeout {{$monitor.Timeout}}
		}
		{{else if eq $monitor.Type "HTTP"}}
		HTTP_GET {
			url {
				path {{$monitor.URLPath}}
				status_code {{$monitor.StatusCode}}
			}
			connect_timeout {{$monitor.Timeout}}
			nb_get_retry {{$monitor.MaxRetries}}
			delay_before_retry {$monitor.Delay}}
		}
		{{else}}
		TCP_CHECK {
			connect_timeout {{$monitor.Timeout}}
			retry {{$monitor.MaxRetries}}
			delay_before_retry {{$monitor.Delay}}
		}
		{{end}}
	}
{{end}}
}
`

var nginxTmpl = `
{{$prot := .Protocol}}
{{$monitor := .Monitor}}
{{range $upstream := .Pools}}
upstream {{$upstream.Name}} {
	{{if ne $upstream.Strategy "round_robin"}}{{$upstream.Strategy}}{{end}}
	{{if and $upstream.SessionStick.Enable (eq $prot "HTTP")}}
	{{$name := $upstream.SessionStick.Name}}
	sticky {{if ne $name ""}}name={{$name}}{{end}};
	{{end}}
	{{range $server := $upstream.Servers}}
	server {{$server.IP}}:{{$server.Port}} {{if $server.Weight}}weight={{$server.Weight}}{{end}} {{if $monitor.MaxRetries}}max_fails={{$monitor.MaxRetries}}{{end}} {{if $monitor.Timeout}}fail_timeout={{$monitor.Timeout}}{{end}};
	{{end}}
}
{{end}}
{{$vip := .VIP}}
{{$port := .Port}}
{{$name := .Name}}
{{$localtions := .Locations}}
server {
	listen {{if ne $vip ""}}{{$vip}}:{{end}}{{$port}} {{if eq $prot "UDP"}}udp{{end}} reuseport;

	{{if eq $prot "HTTP"}}
	access_log /var/log/nginx/{{$name}}-{{$port}}.log  main buffer=32k flush=5s;
	root /usr/share/nginx/html;
	{{range $location := $localtions}}
	location {{$location.URIPath}} {
		proxy_pass http://{{$location.PoolName}};
		proxy_set_header X-Real-IP $remote_addr;
		proxy_set_header X-Forwarded-For  $proxy_add_x_forwarded_for;
		proxy_set_header Host $http_host;

	}
	{{end}}
	error_page 404 /404.html;
		location = /40x.html {
	}

	error_page 500 502 503 504 /50x.html;
		location = /50x.html {
	}
	{{else}}
	{{range $location := $localtions}}
	proxy_pass {{$location.PoolName}};
	{{end}}
	{{end}}
}
`
