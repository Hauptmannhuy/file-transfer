
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
#define MAIN_WINDOW_HEIGHT 600
#define MAIN_WINDOW_WIDTH 800

bool address_panel_enabled;
char current_rendering_address[50] = {};

int text_width(mu_Font font, const char *str, int len) {
  return MeasureText(TextFormat("%.*s", len, str), FONT_SIZE);
}

int text_height(mu_Font font) { return FONT_HEIGHT; }

void init_rendering() {
  SetTargetFPS(60);
  InitWindow(MAIN_WINDOW_WIDTH, MAIN_WINDOW_HEIGHT, "Demo window");
}

void render_host_addr(data_context_t *data_context, mu_Context *ctx) {
  char *host = NULL;
  if (data_context->host_addr != NULL && strlen(data_context->host_addr) > 0) {
    char *host_label = "Host address";
    const int padding_bytes = 2;
    size_t str_lenth = strlen(host_label) + strlen(data_context->host_addr);
    char *host = malloc(str_lenth + padding_bytes);
    sprintf(host, "%s %s", host_label, data_context->host_addr);
    mu_label(ctx, host);
    free(host);
  }
}

void render_address_panel(mu_Context *ctx, char *addr) {
  if (mu_begin_window(ctx, addr, mu_rect(100, 100, 300, 300))) {
    mu_label(ctx, "you opened new window from addr button");
    mu_end_window(ctx);
  }
}

void render_peer_addresses(data_context_t *data_context, mu_Context *ctx) {

  for (int i = 0; i < data_context->addr_count; i++) {
    ip_addr addr = data_context->addrs_buffer[i];

    if (address_panel_enabled) {
      render_address_panel(ctx, current_rendering_address);
    } else {
      if (mu_button(ctx, addr)) {
        strcpy(current_rendering_address, addr);
        address_panel_enabled = true;
      }
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

  while (!WindowShouldClose()) {
    proccess_message_queue(data_context, ipc->message_queue, tpool);

    BeginDrawing();
    ClearBackground(BLACK);

    int mouse_x = GetMouseX();
    int mouse_y = GetMouseY();
    mu_input_mousemove(ctx, mouse_x, mouse_y);

    if (IsMouseButtonPressed(MOUSE_BUTTON_LEFT)) {
      mu_input_mousedown(ctx, mouse_x, mouse_y, MU_MOUSE_LEFT);
    }

    if (IsMouseButtonReleased(MOUSE_BUTTON_LEFT)) {
      mu_input_mouseup(ctx, mouse_x, mouse_y, MU_MOUSE_LEFT);
    }

    if (IsKeyPressed(KEY_F1)) {
      int res = shm_unlink(FILE_NAME);
      u_logger_info("shutting down... \n clearing shared memory: %d\n", res);
      fflush(stdout);
      abort();
    }

    mu_begin(ctx);
    if (mu_begin_window(
            ctx, "My Window",
            mu_rect(10, 10, MAIN_WINDOW_WIDTH / 2, MAIN_WINDOW_HEIGHT / 2))) {
      mu_layout_row(ctx, 2, (int[]){60, -1}, 0);

      mu_label(ctx, "First:");
      if (mu_button(ctx, "Request ip adresses")) {
        command_message cmd_message = {.command_type = CMD_GET_IP_ADDRS,
                                       .payload_size = 0};
        send_ipc_command(cmd_message, ipc);
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

      render_host_addr(data_context, ctx);
      render_peer_addresses(data_context, ctx);
      mu_end_window(ctx);
    }

    if (mu_begin_window(
            ctx, "example address",
            mu_rect(MAIN_WINDOW_HEIGHT / 2, MAIN_WINDOW_WIDTH / 2, 150, 150))) {
      mu_label(ctx, "you opened new window from addr button");
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
        DrawText(cmd->text.str, cmd->text.pos.x, cmd->text.pos.y, FONT_SIZE,
                 cast_color(cmd->text.color));
      } break;
      case MU_COMMAND_CLIP: {
        int h = cmd->clip.rect.h;
        int w = cmd->clip.rect.w;
        int x = cmd->clip.rect.x;
        int y = cmd->clip.rect.y;
        BeginScissorMode(x, y, w, h);
        EndScissorMode();
      }
      }
    }
    EndDrawing();
  }
}
