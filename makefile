compile:
	go build main.go
	sudo setcap cap_net_raw+ep ./main

compile_debug:
	go build -gcflags "all=-N -l" -o debug main.go
	sudo setcap cap_net_raw+ep ./debug


GO_BIN := main
GO_SRC := main.go


CC      := gcc
CFLAGS  := -O2 -Wall
LDFLAGS := -Wl,-rpath,'$$ORIGIN/lib'
LIBS    := -lraylib -lGL -lm -lpthread -ldl -lrt -lX11

SRC := \
	gui/main.c \
	gui/ipc.c \
	gui/logger.c \
	gui/tpool.c \
	gui/data_context.c \
	gui/dependencies/libtinyfiledialogs/tinyfiledialogs.c \
	gui/dependencies/microui/src/microui.c

GUI_BIN := out


DIST := dist

.PHONY: all go gui dist clean


dist: $(DIST)

$(DIST): go gui
	mkdir -p $(DIST)/lib
	mkdir -p $(DIST)/gui

	cp $(GUI_BIN) $(DIST)/gui/
	cp $(GO_BIN)  $(DIST)/
	cp /usr/lib/libraylib.so* $(DIST)/lib/
	zip -r "dist.zip" dist/
	rm -r dist/



go: $(GO_BIN)

$(GO_BIN): $(GO_SRC)
	CGO_ENABLED=0 go build -o $@ $<
	sudo setcap cap_net_raw+ep $@

gui: $(GUI_BIN)

$(GUI_BIN): $(SRC)
	$(CC) $(CFLAGS) $^ $(LDFLAGS) $(LIBS) -o $@


clean:
	rm -rf $(GO_BIN) $(GUI_BIN) $(DIST)
