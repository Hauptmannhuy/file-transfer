#include "data_context.h"
#include "ipc.h"
#include "logger.h"
#include "raylib.h"
#include "stdio.h"
#include "stdlib.h"
#include "tpool.h"


#include "dependencies/microui/src/microui.h"
#include "dependencies/libtinyfiledialogs/tinyfiledialogs.h"

#define FONT_HEIGHT 10
#define FONT_SIZE 10
#define MAIN_WINDOW_HEIGHT 600
#define MAIN_WINDOW_WIDTH 800

bool address_panel_enabled;
char selected_addr[50] = {};
char selected_file_path[100];
mu_Container *current_opened_address_panel;

// TODO testing

int text_width(mu_Font font, const char *str, int len) {
  return MeasureText(TextFormat("%.*s", len, str), FONT_SIZE);
}

int text_height(mu_Font font) { return FONT_HEIGHT; }

void open_file_dialog() {
  
  const char *selected_file = tinyfd_openFileDialog("File explorer", NULL, 0, NULL, NULL, 0);
  strcpy(selected_file_path, selected_file);
  u_logger_info("selected_file_path: %s", selected_file_path);
}

void send_file_path(void *ipc) {
  open_file_dialog();
  while (strlen(selected_file_path) == 0)
    ;
  command_message cmd_msg = {0};
  cmd_msg.command_type = CMD_SEND_FILE_PATH;
  cmd_msg.payload = selected_file_path;
  cmd_msg.payload_size = strlen(selected_file_path);
  send_ipc_command(cmd_msg, ipc);
}

void init_rendering() {
  SetTargetFPS(60);
  InitWindow(MAIN_WINDOW_WIDTH, MAIN_WINDOW_HEIGHT, "Demo Window");
}

void open_panel(mu_Context *ctx, char *id_container_to_open) {
  mu_Container *panel = mu_get_container(ctx, id_container_to_open);
  if (panel == NULL) {
    return;
  }
  u_logger_info("open panel");
  panel->open = 1;
}

void render_host_addr(data_context_t *data_context, mu_Context *ctx) {
  char *host = NULL;
  if (data_context->host_addr != NULL && strlen(data_context->host_addr) > 0) {
    char *host_label = "Host address";
    const int padding_bytes = 2;
    size_t str_lenth = strlen(host_label) + strlen(data_context->host_addr);
    char host[str_lenth + padding_bytes];
    sprintf(host, "%s %s", host_label, data_context->host_addr);
    mu_label(ctx, host);
  }
}

void render_address_panel(mu_Context *ctx, thread_pool_t *tpool,
                          ipc_state_t *ipc, char *addr, data_context_t *data_context) {
  if (address_panel_enabled) {
    if (mu_begin_window(ctx, addr, mu_rect(100, 100, 300, 300))) {
      mu_Container *selected_container = mu_get_container(ctx, addr);
      if (is_connection_established(data_context, addr) == 1) {
        if (mu_button(ctx, "send file")) {
          tpool_add_work(tpool, send_file_path, ipc);
        }
      } else {
        if (mu_button(ctx, "establish connection")) {
          request_p2p(ipc);
        }
      }
      mu_end_window(ctx);
    }
  }
}

void render_peer_addresses_selection(data_context_t *data_context,
                                     mu_Context *ctx) {
  for (int i = 0; i < data_context->addr_count; i++) {
    conn_peer_t *current_addr = data_context->addrs_buffer[i];
    if (mu_button(ctx, current_addr->ip)) {
      open_panel(ctx, current_addr->ip);
      address_panel_enabled = true;
      strcpy(selected_addr, current_addr->ip);
    }
  }
}

Color cast_color(mu_Color color) { return *(Color *)&color; }



int main(int argc, char *argv[]) {
  ipc_state_t *ipc = initialize_shared_memory(argv + 1);
  if (ipc == NULL) {
    u_logger_error("ERROR: error initializing ipc\n");
    return 1;
  }

  mu_Context *ctx = malloc(sizeof(mu_Context));
  data_context_t *data_context = data_context_init();
  mu_init(ctx);
  ctx->text_width = text_width;
  ctx->text_height = text_height;
  init_rendering();
  thread_pool_t *tpool = create_tpool(2);

  start_listener(ipc, tpool);
  char *ip_addrs_buffer = malloc(256);

  SetTargetFPS(60);
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

      if (mu_button(ctx, "Request ip adresses")) {
        get_local_network_hosts(ipc);
        u_logger_info("Button1 pressed\n");
      }


      render_host_addr(data_context, ctx);
      render_peer_addresses_selection(data_context, ctx);
      render_address_panel(ctx, tpool, ipc, selected_addr, data_context);

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
        // BeginScissorMode(x, y, w, h);
        // EndScissorMode();
      }
      }
    }

    EndDrawing();
  }
  return 0;
}
