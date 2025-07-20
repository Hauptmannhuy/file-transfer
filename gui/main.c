#include "microui.h"
#include "stdio.h"
#include "memory.h"
#include "stdlib.h"
#include "raylib.h"

int FONT_SIZE = 28;
mu_Font *DEFAULT_FONT = NULL; 

int text_height(mu_Font font){

    // return MeasureText(font, FONT_SIZE);
}

int text_width(mu_Font font,const char *text , int len) {
    printf("text pointer %p\n", text);
    printf("font (%p)\n", font);
    printf("len %d\n", len);
    // if (text != NULL) {
    //     printf("text %s\n", *text);
    // }

    // return MeasureText(text, font);
}







void process_ui(mu_Context *ctx) {
    BeginDrawing();
    if (mu_begin_window(ctx, "Demo Window", mu_rect(40, 40, 300, 450))) {
            // mu_Container *win = mu_get_current_container(ctx);
            // win->rect.w = mu_max(win->rect.w, 240);
            // win->rect.h = mu_max(win->rect.h, 300);
            
            /* window info */
            // if (mu_header(ctx, "Window Info")) {
            //     mu_Container *win = mu_get_current_container(ctx);
            //     char buf[64];
            //     mu_layout_row(ctx, 2, (int[]) { 54, -1 }, 0);
            //     mu_label(ctx,"Position:");
            //     sprintf(buf, "%d, %d", win->rect.x, win->rect.y); mu_label(ctx, buf);
            //     mu_label(ctx, "Size:");
            //     sprintf(buf, "%d, %d", win->rect.w, win->rect.h); mu_label(ctx, buf);
            // }
            mu_label(ctx, "First:");
            if (mu_button(ctx, "Button1")) {
                printf("Button1 pressed\n");
            }

            mu_label(ctx, "Second:");
            if (mu_button(ctx, "Button2")) {
                mu_open_popup(ctx, "My Popup");
            }            
                mu_end_window(ctx);
    }
    EndDrawing();
}

void init_rendering() {
    char * window_name = "Demo Window";
    InitWindow(600, 600, window_name);

}

void render_rect(mu_Rect rect, mu_Color color) {
   Color rgba = {
        color.r, color.g, color.b, color.a
    };
    DrawRectangle(rect.x, rect.y, rect.w, rect.h, rgba);
}

static void loop(mu_Context *ctx) {
    

    while (!WindowShouldClose()) {
        PollInputEvents();

        if (IsKeyDown(KEY_SPACE)) {
            printf("space is pressed\n");
        }
        
        


        mu_begin(ctx);
        process_ui(ctx);
        mu_end(ctx);

        mu_Command *cmd = NULL;
        while (mu_next_command(ctx, &cmd) > 0) {
            if (cmd->type == MU_COMMAND_TEXT) {
                printf("MU_COMMAND_TEXT\n");
                // render_text(cmd->text.font, cmd->text.text, cmd->text.pos.x, cmd->text.pos.y, cmd->text.color);
              }
              if (cmd->type == MU_COMMAND_RECT) {
                printf("MU_COMMAND_RECT\n");
                render_rect(cmd->rect.rect, cmd->rect.color);
              }
              if (cmd->type == MU_COMMAND_ICON) {
                printf("MU_COMMAND_ICON\n");
                // render_icon(cmd->icon.id, cmd->icon.rect, cmd->icon.color);
              }
              if (cmd->type == MU_COMMAND_CLIP) {
                printf("MU_COMMAND_CLIP\n");
                // set_clip_rect(cmd->clip.rect);
              }
        }

        // abort();
    }
}

void load_font(mu_Context *ctx) {
    Font loaded_fonts = LoadFont("fonts.ttf");
    DEFAULT_FONT = malloc(sizeof((mu_Font)&loaded_fonts));

    ctx->style->font = DEFAULT_FONT;
}



int main(){
    mu_Context *ctx = malloc(sizeof(mu_Context));
    mu_init(ctx);
    
    ctx->text_height = text_height;
    ctx->text_width = text_width;
    load_font(ctx);
    init_rendering();


    loop(ctx);
}


// void process_ui_logic(mu_Context *ctx) {
//    mu_begin(ctx);
//    mu_end(ctx);
// }




