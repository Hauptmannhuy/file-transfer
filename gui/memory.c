#include "memory.h"




shared_memory* initialize_shared_memory(){
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
    uintptr_t start_addres = (uintptr_t)addr;
    shared_memory *memory = malloc(sizeof(shared_memory));
    memory->memory_block = memory_block;
    memory->start_adress = start_addres;

    return memory;
}


int client_command_to_addr(int cmd) {
    switch (cmd)
    {
    case MEM_GET_IP_ADDRS:
         {
            return CMD_ADDR_LOC;
         } break;
    default:
         return -1;
        break;
    }
}

void send_ipc_command(int cmd, shared_memory *ipcState) {
    int translated_command_addr = client_command_to_addr(cmd);
    if (translated_command_addr == -1) { printf("unknown ipc command\n"); exit(EXIT_FAILURE); } 
    ipcState->memory_block[translated_command_addr] = 1;
}

void initialize_shared_memory_state(void *addr_ptr) {
    
}

shared_memory* initialize_memory() {

}


