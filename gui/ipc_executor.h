#include "memory.h"


typedef struct ipc_command {
    int command_type;
    char *memory;
    void *pointer_to_cmd;
    void *( *handler)(void *);
    
} ipc_command;

typedef struct ipc_get_addresses_command {
    char *buffer;
    char *memory_ptr;
} ipc_get_addresses_command;


void new_worker(int type, shared_memory *memory, char *buffer);