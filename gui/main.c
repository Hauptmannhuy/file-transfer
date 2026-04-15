#include "data_context.h"
#include "ipc.h"
#include "logger.h"
#include "raylib.h"
#include "stdio.h"
#include "stdlib.h"
#include "tpool.h"

#include "dependencies/libtinyfiledialogs/tinyfiledialogs.h"
#include "dependencies/microui/src/microui.h"
#include "stdbool.h"
#include "utils.h"
#include <string.h>

#define FONT_HEIGHT 10
#define FONT_SIZE 10

bool address_panel_enabled;
char selected_addr[50] = {};
char selected_file_path[100];
bool copy_pasted;

mu_Container *current_opened_address_panel;

typedef struct {
  int buffer_size;
  int count;
  char *buffer;
} text_input_buffer;

// TODO testing

int text_width(mu_Font font, const char *str, int len) {
  return MeasureText(TextFormat("%.*s", len, str), FONT_SIZE);
}

int text_height(mu_Font font) { return FONT_HEIGHT; }

void handle_input(text_input_buffer *text_buffer) {
  char key = (char)GetKeyPressed();
  if (key != 0 && text_buffer->count < text_buffer->buffer_size - 1 &&
      ((key | 0x20) >= 97 && 122 >= (key | 0x20) || (key >= 48 && key <= 57) ||
       (key == 46)) &&
      !IsKeyDown(KEY_LEFT_CONTROL)) {

    text_buffer->buffer[text_buffer->count] = key;
    text_buffer->buffer[text_buffer->count + 1] = '\0';
    text_buffer->count += 1;
  } else if ((IsKeyPressed(KEY_BACKSPACE) || IsKeyDown(KEY_BACKSPACE)) &&
             text_buffer->count >= 0) {
    text_buffer->buffer[text_buffer->count] = '\0';
    if (text_buffer->count > 0) {
      text_buffer->count -= 1;
    }
  }

  if (!copy_pasted && IsKeyDown(KEY_LEFT_CONTROL) && IsKeyDown(KEY_V)) {
    const char *clipboard_text = GetClipboardText();
    int clipboard_length = strlen(clipboard_text);
    char *text_buffer_pointer = &text_buffer->buffer[text_buffer->count];
    memcpy(text_buffer_pointer, clipboard_text, clipboard_length + 1);
    text_buffer->count += clipboard_length;
    copy_pasted = true;
  }

  if (copy_pasted &&
      (IsKeyReleased(KEY_LEFT_CONTROL) || IsKeyReleased(KEY_V))) {
    copy_pasted = !copy_pasted;
  }
}

void open_file_dialog() {

  const char *selected_file =
      tinyfd_openFileDialog("File explorer", NULL, 0, NULL, NULL, 0);
  strcpy(selected_file_path, selected_file);
  u_logger_info("selected_file_path: %s", selected_file_path);
}

void send_file_path(void *ipc) {

  // open_file_dialog();
  // while (strlen(selected_file_path) == 0)
  //   ;
  // command_message cmd_msg = {0};
  // cmd_msg.command_type = CMD_SEND_FILE_PATH;
  // cmd_msg.json_payload = selected_file_path;
  // cmd_msg.payload_size = strlen(selected_file_path);
  // send_ipc_command(cmd_msg, ipc);
}

void init_rendering(int width, int height) {
  SetConfigFlags(FLAG_WINDOW_HIGHDPI);
  SetTargetFPS(170);
  InitWindow(width, height, "Demo Window");
  int monitor = GetCurrentMonitor();
  SetWindowSize(GetMonitorWidth(monitor) / 2, GetMonitorHeight(monitor) / 2);
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
    mu_layout_row(ctx, 2, (int[]){150, 100}, 0);
    char *host_label = "Host address";
    const int padding_bytes = 2;
    size_t str_lenth = strlen(host_label) + strlen(data_context->host_addr);
    char host[str_lenth + padding_bytes];
    sprintf(host, "%s %s", host_label, data_context->host_addr);
    mu_label(ctx, host);
    if (mu_button(ctx, "Copy to clipboard")) {
      SetClipboardText(data_context->host_addr);
    }
  }
}

int ask_user_connection_response(mu_Context *ctx) {
  int response = 0;
  if (mu_button(ctx, "Accept connection")) {
    response = 1;
  } else if (mu_button(ctx, "Refuse connection")) {
    response = 2;
  }
  return response;
}

void render_address_panel(mu_Context *ctx, thread_pool_t *tpool,
                          ipc_state_t *ipc, char *addr,
                          data_context_t *data_context) {
  if (address_panel_enabled) {
    if (mu_begin_window(ctx, addr, mu_rect(100, 100, 300, 300))) {
      if (is_connection_established(data_context, addr) == 1) {
        if (mu_button(ctx, "send file")) {
          tpool_add_work(tpool, send_file_path, ipc);
        }
      } else {
        if (is_connection_pending(data_context, addr)) {
          int response = ask_user_connection_response(ctx);
          if (response > 0) {
            accept_p2p(ipc, addr, data_context, response);
          }
        } else {
          if (mu_button(ctx, "establish connection")) {
            request_p2p(ipc, addr);
          }
        }
      }
      mu_end_window(ctx);
    }
  }
}

void render_peer_addresses_selection(data_context_t *data_context,
                                     mu_Context *ctx) {
  Array_header_t *header = get_header(data_context->local_peers_dynamic_array);
  for (int i = 0; i < header->count; i++) {
    conn_peer_t *current_addr = data_context->local_peers_dynamic_array[i];
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
  int text_input_buffer_size = 300;
  char _text_buffer[text_input_buffer_size] = {};
  text_input_buffer text_buffer = {.buffer_size = text_input_buffer_size,
                                   .count = 0,
                                   .buffer = _text_buffer};

  mu_init(ctx);
  ctx->text_width = text_width;
  ctx->text_height = text_height;
  init_rendering(0, 0);
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
    if (mu_begin_window(ctx, "My Window", mu_rect(10, 10, 800, 600))) {

      mu_layout_row(ctx, 2, (int[]){300, 100}, 0);
      mu_textbox(ctx, text_buffer.buffer, text_input_buffer_size);
      if (ctx->last_id == ctx->focus) {
        handle_input(&text_buffer);
      }

      if (mu_button(ctx, "Connect")) {
        if (strlen(text_buffer.buffer) > 0) {
          request_p2p(ipc, text_buffer.buffer);
        }
      }

      mu_layout_row(ctx, 1, (int[]){300, 100}, 0);
      if (mu_button(ctx, "Request ip adresses")) {
        get_local_network_hosts(ipc);
      }

      render_host_addr(data_context, ctx);
      render_peer_addresses_selection(data_context, ctx);
      render_address_panel(ctx, tpool, ipc, selected_addr, data_context);

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
