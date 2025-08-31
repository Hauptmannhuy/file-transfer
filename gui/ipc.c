#include "ipc.h"
#include <pthread.h>
#include <inttypes.h>




void* worker(void *arg) {
    struct ipc_command *command = (struct ipc_command*)arg;
    printf("INFO: STARTED NEW WORKER\n");

    int status = check_rw_status(command->ipc_state_t);
    while (status == -1) {
        status = check_rw_status(command->ipc_state_t);
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

void handle_message(int cmd_type, ipc_state_t *ipc, char *buffer) {
    pthread_t thread;

    struct ipc_command *command = malloc(sizeof(struct ipc_command));
    if (!command) return; 

    void *(*handler)(void *);
    command->ipc_state_t = ipc;
    switch (cmd_type)
    {
    case MEM_GET_IP_ADDRS:
    struct ipc_get_addresses_command *cmd = malloc(sizeof(ipc_get_addresses_command));
    cmd->buffer = buffer;
    cmd->memory_ptr = ipc->memory;

        command->pointer_to_cmd = cmd;
        command->handler = get_addresses;
        break;
    
    default:
        break;
    }

    pthread_create(&thread, NULL, worker, (void*)command);
    pthread_detach(thread); 
}




void copy_to_buff(char *memory, char *buffer) {
    // int offset = CMD_MESSAGE_VALUE_ADRESS_START;
    // int message_length = memory[offset];
    // int buffer_offset = 0;

    // while (message_length > 0)
    // {
    //     memcpy(buffer + buffer_offset, memory + offset + 1, message_length);
    //     buffer[buffer_offset + message_length] = '\n';
    //     offset += message_length+1;
    //     buffer_offset += message_length;
    //     message_length = memory[offset];
    // }
    // buffer[buffer_offset+message_length] = '\0';
    // printf("buffer: %s\n", buffer);
}


void send_ipc_command(command_message cmdMsg, ipc_state_t *ipcState) {
    uint32_t write_offset = *ipcState->back_cb->write_offset;
    char *destination = ipcState->back_cb->memory_block + write_offset;
    int cmd_type_size = sizeof(cmdMsg.command_type);
    printf("size %d\n", cmd_type_size);
    printf("write offset %d\n", write_offset);
    memcpy(destination, &cmdMsg.command_type, cmd_type_size);
    // copy payload size as the same size as cmd type as they both uint_32t
    memcpy(destination+cmd_type_size, &cmdMsg.payload_size, cmd_type_size);
}

int check_rw_status(ipc_state_t *ipc_state){
    int rdwr_offset = ipc_state->back_cb->rdwr_status;
    if (ipc_state->back_cb->memory_block[rdwr_offset] == READY_RDWR) return 0;
    if (ipc_state->back_cb->memory_block[rdwr_offset] == ACTIVE_RDWR) return -1;
};


ipc_state_t* initialize_shared_memory(){
    int fd = shm_open(FILE_NAME, O_CREAT | O_RDWR, 0666);
    if (fd == -1) {
        printf("IPC INFO: error shm_open, %d\n", fd);
        return NULL;
    }
    
    ftruncate(fd, ADRESS_SPACE_SIZE);

    if (fd == -1){ printf("IPC INFO: failed to create shared memory\n"); return NULL; };
    printf("IPC INFO: shared memory created\n");

    void *addr = mmap(NULL, ADRESS_SPACE_SIZE, PROT_READ | PROT_WRITE, MAP_SHARED, fd, 0);
    if (addr == NULL) {
        return NULL;
    }
    printf("IPC INFO: key_shm_open %d\n", fd);
    printf("IPC INFO: addr %p\n", addr);
    
    char *memory_block = (char*)addr;

    ipc_state_t *ipc = malloc(sizeof(ipc_state_t));
    
    control_block *back_cb = malloc(sizeof(control_block));
    control_block *front_cb = malloc(sizeof(control_block));

    back_cb->memory_block = memory_block + Bblock_addr_space;
    front_cb->memory_block = memory_block + Fblock_addr_space;
    
    int offset_start = 10;

    uint32_t *ptr_start_read_offset_f  = (uint32_t*)(front_cb->memory_block); 
    uint32_t *ptr_start_write_offset_f = (uint32_t*)front_cb->memory_block+1;        

    *ptr_start_write_offset_f = offset_start;
    *ptr_start_read_offset_f  = offset_start;

    uint32_t *ptr_start_read_offset_b  = (uint32_t*)(back_cb->memory_block);
    uint32_t *ptr_start_write_offset_b = (uint32_t*)back_cb->memory_block+1;

    *ptr_start_write_offset_b = offset_start;
    *ptr_start_read_offset_b  = offset_start;

    

    back_cb->read_offset = ptr_start_read_offset_b;
    back_cb->write_offset = ptr_start_write_offset_b;

    front_cb->read_offset = ptr_start_read_offset_f;
    front_cb->write_offset = ptr_start_write_offset_f;

    ipc->back_cb = back_cb;
    ipc->front_cb = front_cb;
    ipc->memory = memory_block;
    return ipc;
}

