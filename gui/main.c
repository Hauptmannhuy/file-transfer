
#include "microui/src/microui.h"
#include "raylib.h"
#include "stdio.h"
#include "stdlib.h"

#define FONT_HEIGHT 10
#define FONT_SIZE 10
#define WINDOW_HEIGHT 600
#define WINDOW_WIDTH 800


int text_width(mu_Font font, const char* str, int len){
    return MeasureText(TextFormat("%.*s", len, str), FONT_SIZE);
}

int text_height(mu_Font font) {
    return FONT_HEIGHT;
}

void init_rendering() {
    SetTargetFPS(60);
    InitWindow(WINDOW_WIDTH, WINDOW_HEIGHT, "Demo window");
}

Color cast_color(mu_Color color){
   return *(Color*)&color;
}



void main(){
    mu_Context *ctx = malloc(sizeof(mu_Context));
    mu_init(ctx);
    ctx->text_width = text_width;
    ctx->text_height = text_height;
    init_rendering();

    while (!WindowShouldClose())
    {
        BeginDrawing();

        
        if (IsMouseButtonDown(MOUSE_LEFT_BUTTON)){
             Vector2 position = GetMousePosition();
            mu_input_mousedown(ctx, position.x, position.y, MU_MOUSE_LEFT);
        }

        if (IsMouseButtonUp(MOUSE_LEFT_BUTTON)) {
            Vector2 position = GetMousePosition();
            mu_input_mouseup(ctx, position.x, position.y, MU_MOUSE_LEFT);
        }

        mu_begin(ctx);
        if (mu_begin_window(ctx, "My Window", mu_rect(10, 10, 800, 600))) {
        mu_layout_row(ctx, 2, (int[]) { 60, -1 }, 0);

        mu_label(ctx, "First:");
        if (mu_button(ctx, "Button1")) {
            printf("Button1 pressed\n");
        }

        mu_label(ctx, "Second:");
        if (mu_button(ctx, "Button2")) {
            mu_open_popup(ctx, "My Popup");
        }

        if (mu_begin_popup(ctx, "My Popup")) {
            mu_label(ctx, "Hello world!");
            mu_end_popup(ctx);
        }
        mu_end_window(ctx);
        }

        mu_end(ctx);

        mu_Command *cmd = 0;
        mu_next_command(ctx, &cmd);
        while(mu_next_command(ctx, &cmd)){
             switch (cmd->type) {
                case MU_COMMAND_RECT:
                {
                    DrawRectangle(cmd->rect.rect.x, cmd->rect.rect.y, cmd->rect.rect.w, cmd->rect.rect.h, cast_color(cmd->rect.color));
                } break;
                case MU_COMMAND_TEXT:
                {
                    // printf("NK_COMMAND_TEXT %s\n", cmd->text.str);
                    DrawText(cmd->text.str, cmd->text.pos.x, cmd->text.pos.y, FONT_SIZE, cast_color(cmd->text.color));
                } break;
                case MU_COMMAND_CLIP:
                {
                    printf("NK_COMMAND_CLIP\n");
                }
            }
        
        }
       
        ClearBackground(BLACK);
        EndDrawing();
    }

}




