
#include "data_context.h"
#include "ipc.h"
#include "logger.h"
#include "microui/src/microui.h"
#include "raylib.h"
#include "stdio.h"
#include "stdlib.h"
#include "tpool.h"

#define FONT_HEIGHT 10
#define FONT_SIZE 10
#define WINDOW_HEIGHT 800
#define WINDOW_WIDTH 800

int text_width(mu_Font font, const char *str, int len) {
  return MeasureText(TextFormat("%.*s", len, str), FONT_SIZE);
}

int text_height(mu_Font font) { return FONT_HEIGHT; }

void init_rendering() {
  SetTargetFPS(60);
  InitWindow(WINDOW_WIDTH, WINDOW_HEIGHT, "Demo window");
}

void render_addresses(data_context_t *data_context, mu_Context *ctx) {
  char *host = NULL;
  if (data_context->host_addr != NULL && strlen(data_context->host_addr) > 0) {
    char *host_label = "Host address";
    size_t size = strlen(host_label) + strlen(data_context->host_addr);
    char *host = malloc(size + 1);
    sprintf(host, "%s %s", host_label, data_context->host_addr);
    mu_label(ctx, host);
    free(host);
  }
  
  for (int i = 0; i < data_context->addr_count; i++) {
    ip_addr addr = data_context->addrs_buffer[i];
    if (mu_button(ctx, addr)) {
      mu_open_popup(ctx, addr);
      
    }
    if (mu_begin_popup(ctx, addr)) {
        if (mu_begin_window(ctx, addr, mu_rect(100, 100, 300,300))) {
          mu_label(ctx, "you opened new window from addr button");
          mu_end_window(ctx);
        }

        mu_end_popup(ctx);
    }
  }
}

Color cast_color(mu_Color color) { return *(Color *)&color; }

void main() {
  ipc_state_t *ipc = initialize_shared_memory();
  if (ipc == NULL) {
    u_logger_error("ERROR: error initializing ipc\n");
    return;
  }

  mu_Context *ctx = malloc(sizeof(mu_Context));
  data_context_t *data_context = data_context_init();
  mu_init(ctx);
  ctx->text_width = text_width;
  ctx->text_height = text_height;
  init_rendering();
  thread_pool_t *tpool = create_tpool(4);

  start_listener(ipc, tpool);
  char *ip_addrs_buffer = malloc(256);

  bool is_left_mouse_down;

  while (!WindowShouldClose()) {
    proccess_message_queue(data_context, ipc->message_queue, tpool);

    BeginDrawing();
    if (IsMouseButtonReleased(MOUSE_LEFT_BUTTON)) {
      is_left_mouse_down = false;
    }

    if (IsMouseButtonDown(MOUSE_LEFT_BUTTON) && !is_left_mouse_down) {
      Vector2 position = GetMousePosition();
      mu_input_mousedown(ctx, position.x, position.y, MU_MOUSE_LEFT);
      is_left_mouse_down = true;
    }

    if (IsMouseButtonUp(MOUSE_LEFT_BUTTON)) {
      Vector2 position = GetMousePosition();
      mu_input_mouseup(ctx, position.x, position.y, MU_MOUSE_LEFT);
    }

    if (IsKeyPressed(KEY_F1)) {
      int res = shm_unlink(FILE_NAME);
      u_logger_info("shutting down... \n clearing shared memory: %d\n", res);
      fflush(stdout);
      abort();
    }

    mu_begin(ctx);
    if (mu_begin_window(ctx, "My Window", mu_rect(10, 10, WINDOW_WIDTH, WINDOW_HEIGHT))) {
      mu_layout_row(ctx, 3, (int[]){60, -1}, 0);

      mu_label(ctx, "First:");
      if (mu_button(ctx, "Request ip adresses")) {
        command_message cmd = {.command_type = CMD_GET_IP_ADDRS,
                               .payload_size = 0};
        send_ipc_command(cmd, ipc);
        u_logger_info("Button1 pressed\n");
      }

      mu_label(ctx, "Second:");
      if (mu_button(ctx, "Button2")) {
        mu_open_popup(ctx, "My Popup");
      }

      if (mu_begin_popup(ctx, "My Popup")) {
        mu_label(ctx, "Hello world!");
        mu_end_popup(ctx);
      }

      render_addresses(data_context, ctx);

      mu_end_window(ctx);
    }

    mu_end(ctx);

    mu_Command *cmd = 0;
    mu_next_command(ctx, &cmd);
    while (mu_next_command(ctx, &cmd)) {
      switch (cmd->type) {
      case MU_COMMAND_RECT: {
        DrawRectangle(cmd->rect.rect.x, cmd->rect.rect.y, cmd->rect.rect.w,
                      cmd->rect.rect.h, cast_color(cmd->rect.color));
      } break;
      case MU_COMMAND_TEXT: {
        // u_logger_info("NK_COMMAND_TEXT %s\n", cmd->text.str);
        DrawText(cmd->text.str, cmd->text.pos.x, cmd->text.pos.y, FONT_SIZE,
                 cast_color(cmd->text.color));
      } break;
      case MU_COMMAND_CLIP: {
        // u_logger_info("NK_COMMAND_CLIP\n");
        int h = cmd->clip.rect.h;
        int w = cmd->clip.rect.w;
        int x = cmd->clip.rect.x;
        int y = cmd->clip.rect.y;
        // BeginScissorMode(x,y,w,h);
        u_logger_info("mu_clip: h %d w %d x %d y %d");
        
      }
    }
  }
    ClearBackground(BLACK);
    // EndScissorMode();
    EndDrawing();
  }
}
