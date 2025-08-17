

#pragma once 

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
#define ADRESS_SPACE_SIZE 1024*256


#define CMD_TYPE_MESSAGE_ADRESS 0
#define CMD_STATUS_ADRESS 1
#define CMD_MESSAGE_VALUE_ADRESS_START 2
#define CMD_MESSAGE_VALUE_ADDRESS_END 66

typedef struct {
    char *memory_block;
    uintptr_t start_adress;
} shared_memory;


enum command_types {
    MEM_GET_IP_ADDRS = 1,
    MEM_RECEIVED_IP_ADDRS
};

enum status {
    STATUS_IDLE = 0,
    READY_RDWR = 1,
    ACTIVE_RDWR,
};



void send_ipc_command(int cmd, shared_memory *ipcState);
shared_memory* initialize_shared_memory();