#include "memory.h"


typedef struct ipc_command {
    char *memory;
    int command_type;
    void (*handler)(void*);
} ipc_command;


void new_worker(void (*handler)(void*), shared_memory *memory, int type);