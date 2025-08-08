
#include "sys/mman.h"
#include "sys/stat.h"
#include <sys/types.h>

#include "unistd.h"
#include "stdint.h"
#include "stdlib.h"
#include "stdio.h"
#include "string.h"

#include "fcntl.h"



#define FILE_NAME "/mySharedMem"
#define ADRESS_SPACE_SIZE 4096



typedef struct {
    char *memory_block;
    uintptr_t start_adress;
} shared_memory;


enum commands {
    MEM_GET_IP_ADDRS,
    MEM_RECEIVED_IP_ADDRS
};

enum status {
    READY_RDWR,
    ACTIVE_RDWR,
};

enum addresses {
    CMD_ADDR_LOC = 20,
};


void send_ipc_command(int cmd, shared_memory *ipcState);
shared_memory* initialize_shared_memory();