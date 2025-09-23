#pragma once

#include <sys/mman.h>
#include <sys/stat.h>
#include <sys/types.h>

#include <fcntl.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

#include "data_context.h"
#include "tpool.h"

#define FILE_NAME "/mySharedMem"
#define ADRESS_SPACE_SIZE 1024 * 256

#define Fblock_addr_space 0
#define Bblock_addr_space ADRESS_SPACE_SIZE / 2
#define CONTROL_BLOCK_SIZE 16
#define message_queue_capacity 256

// TODO: think about how to pass user data to back
typedef struct {
  uint32_t command_type;
  uint32_t payload_size;
  char *payload;
} command_message;

typedef struct {
  uint32_t *read_offset;
  uint32_t *write_offset;
  uint8_t rdwr_status;
  char *memory_block;
} control_block;

typedef struct {
  command_message *buffer;
  int count;
  int capacity;
  int head;
  int tail;

} message_queue_t;

typedef struct {
  char *memory;
  uintptr_t start_adress;
  control_block *back_cb;
  control_block *front_cb;
  message_queue_t *message_queue;
} ipc_state_t;

typedef struct ipc_command {
  command_message command_message;
  ipc_state_t *ipc_state_t;
  void *pointer_to_cmd;
  void *(*handler)(void *);

} ipc_command;

typedef struct ipc_get_addresses_command {
  char *buffer;
  char *memory_ptr;
} ipc_get_addresses_command;

enum command_types { CMD_GET_IP_ADDRS = 1 };

enum status {
  READY_RDWR = 0,
  ACTIVE_RDWR,
};

typedef struct command_handler_t {
  data_context_t *data_context_t;
  thread_func_t func;
  char *buffer;
} command_handler_t;

void start_listener(ipc_state_t *ipc_state, thread_pool_t *tpool);
// void copy_to_buff(char *memory, char *buffer);
int check_rw_status(ipc_state_t *ipc_state);
void send_ipc_command(command_message cmdMsg, ipc_state_t *ipc_state);
void handle_message(int type, ipc_state_t *memory, char *buffer);
ipc_state_t *initialize_shared_memory();

void proccess_message_queue(data_context_t *data_context,
                            message_queue_t *message_queue,
                            thread_pool_t *tpool);

command_handler_t *get_command_handler(data_context_t *data_context,
                                       int cmd_type, char *buffer);