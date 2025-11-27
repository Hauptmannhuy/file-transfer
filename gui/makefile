compile:
	gcc `pkg-config --cflags gtk4` `pkg-config --libs gtk4` main.c ipc.c logger.c tpool.c microui/src/microui.c data_context.c -lraylib -lGL -lm -lpthread -ldl -lrt -lX11 -o out

sanitize-compile:
	gcc main.c ipc.c logger.c tpool.c microui/src/microui.c data_context.c -lraylib -lGL -lm -lpthread -ldl -lrt -lX11 -o out -fsanitize=address
