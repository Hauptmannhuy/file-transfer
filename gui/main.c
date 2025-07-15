#include "../microui/src/microui.h"
#include "stdio.h"
#include "memory.h"
#include "stdlib.h"

int text_height(mu_Font font){
    return sizeof(font);
}

int text_width(mu_Font font,const char *text , int len) {
    return 0;
}




static void loop(mu_Context *ctx) {
    while (1 == 1) {
        if (mu_begin_window(ctx, "Demo Window", mu_rect(40, 40, 300, 450))) {
            mu_Container *win = mu_get_current_container(ctx);
            win->rect.w = mu_max(win->rect.w, 240);
            win->rect.h = mu_max(win->rect.h, 300);
            
            /* window info */
            if (mu_header(ctx, "Window Info")) {
                mu_Container *win = mu_get_current_container(ctx);
                char buf[64];
                mu_layout_row(ctx, 2, (int[]) { 54, -1 }, 0);
                mu_label(ctx,"Position:");
                sprintf(buf, "%d, %d", win->rect.x, win->rect.y); mu_label(ctx, buf);
                mu_label(ctx, "Size:");
                sprintf(buf, "%d, %d", win->rect.w, win->rect.h); mu_label(ctx, buf);
            }
        }
    }
}



int main(){
    mu_Context *ctx = malloc(sizeof(mu_Context));
    mu_init(ctx);
    mu_input_keydown(ctx, 1);
    
    ctx->text_height = text_height;
    ctx->text_width = text_width;
    loop(ctx);
}




