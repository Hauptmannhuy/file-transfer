compile:
	go build main.go
	sudo setcap cap_net_raw+ep ./main

compile_debug:
	go build -gcflags "all=-N -l" -o debug main.go
	sudo setcap cap_net_raw+ep ./debug

run:
	go build main.go
	sudo setcap cap_net_raw+ep ./main	
	gcc `pkg-config --cflags gtk4` `pkg-config --libs gtk4` gui/main.c gui/ipc.c gui/logger.c gui/tpool.c gui/microui/src/microui.c gui/data_context.c -lraylib -lGL -lm -lpthread -ldl -lrt -lX11 -o out         
	./gui/out                                                                                                                                                                             
	./main                                                                                                                                                                                 
