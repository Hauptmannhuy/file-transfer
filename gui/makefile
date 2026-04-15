compile:
	gcc main.c dependencies/libtinyfiledialogs/tinyfiledialogs.c ipc.c logger.c tpool.c utils.c dependencies/microui/src/microui.c dependencies/cjson/cJSON.c data_context.c -lraylib -lGL -lm -lpthread -ldl -lrt -lX11 -o out

sanitize-compile:
	gcc main.c dependencies/libtinyfiledialogs/tinyfiledialogs.c ipc.c logger.c tpool.c utils.c dependencies/microui/src/microui.c dependencies/cjson/cJSON.c data_context.c -lraylib -lGL -lm -lpthread -ldl -lrt -lX11 -o out -fsanitize=address

compile_debug:
	gcc -g main.c dependencies/libtinyfiledialogs/tinyfiledialogs.c ipc.c logger.c tpool.c utils.c dependencies/microui/src/microui.c dependencies/cjson/cJSON.c data_context.c -lraylib -lGL -lm -lpthread -ldl -lrt -lX11 -o debug

CC      := gcc
CFLAGS  := -O2 -Wall
LDFLAGS := -Wl,-rpath,'$$ORIGIN/lib'
LIBS    := -lraylib -lGL -lm -lpthread -ldl -lrt -lX11

SRC := \
	main.c \
	ipc.c \
	logger.c \
	tpool.c \
	data_context.c \
	utils.c \
	dependencies/libtinyfiledialogs/tinyfiledialogs.c \
	dependencies/microui/src/microui.c \
	dependencies/cjson/cJSON.c data_context.c


TARGET := out
DIST   := dist

.PHONY: all clean dist

all: $(TARGET)

$(TARGET): $(SRC)
	$(CC) $(CFLAGS) $^ $(LDFLAGS) $(LIBS) -o $@

dist: $(TARGET)
	mkdir -p $(DIST)/lib
	cp $(TARGET) $(DIST)/
	cp /usr/lib/libraylib.so* $(DIST)/lib/
	

clean:
	rm -rf $(TARGET) $(DIST)