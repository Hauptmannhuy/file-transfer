#include "ipc_executor.h"
#include <pthread.h>





void* worker(void *arg) {
    struct ipc_command *command = (struct ipc_command*)arg;
    printf("INFO: STARTED NEW WORKER\n");

    int status = check_rw_status(command->memory);
    while (status == -1) {
        status = check_rw_status(command->memory);
    }

    command->handler(command->pointer_to_cmd);

    free(command->pointer_to_cmd);
    free(command);

    printf("INFO: worker end\n");
    return NULL;
}

void* get_addresses(void *arg) {
    ipc_get_addresses_command *cmd = (ipc_get_addresses_command*)arg;
    copy_to_buff(cmd->memory_ptr, cmd->buffer); 
    return NULL;
}

void new_worker(int cmd_type, shared_memory *memory, char *buffer) {
    pthread_t thread;

    struct ipc_command *command = malloc(sizeof(struct ipc_command));
    if (!command) return; 

    void *(*handler)(void *);
    command->memory = memory->memory_block;
    switch (cmd_type)
    {
    case MEM_GET_IP_ADDRS:
    struct ipc_get_addresses_command *cmd = malloc(sizeof(ipc_get_addresses_command));
    cmd->buffer = buffer;
    cmd->memory_ptr = memory->memory_block;

        command->pointer_to_cmd = cmd;
        command->handler = get_addresses;
        break;
    
    default:
        break;
    }

    pthread_create(&thread, NULL, worker, (void*)command);
    pthread_detach(thread); 
}


