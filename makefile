run_app:
	go build main.go
	sudo setcap cap_net_raw+ep ./main
	./main

compile_debug:
	go build -gcflags "all=-N -l" -o debug main.go
	sudo setcap cap_net_raw+ep ./debug