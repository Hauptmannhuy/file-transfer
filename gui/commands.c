#include "commands.h"
#include <pthread.h>


int read_server(char *memory) {
    if (memory[CMD_STATUS_ADRESS] == READY_RDWR) return 0;
    if (memory[CMD_STATUS_ADRESS] == ACTIVE_RDWR) return -1;
}

void* worker(void *arg) {
    struct ipc_command *command = (struct ipc_command*)arg;
    printf("INFO: STARTED NEW WORKER\n");

    int status = read_server(command->memory);
    while (status != 0) {
        status = read_server(command->memory);
    }

    command->handler(command->memory);

    free(command); 
    return NULL;
}

void new_worker(void (*handler)(void*), shared_memory *memory, int type) {
    pthread_t thread;

    struct ipc_command *command = malloc(sizeof(struct ipc_command));
    if (!command) return; 

    command->memory = memory->memory_block;
    command->command_type = type;
    command->handler = handler;

    pthread_create(&thread, NULL, worker, (void*)command);
    pthread_detach(thread); 
}


