#include "ipc.h"
#include "logger.h"
#include <inttypes.h>
#include <string.h>

#define uint32_size sizeof(uint32_t)

#define SHIFT_OFFSET(offset_ptr, payload_size)                                 \
  ((*offset_ptr) += (payload_size) + (uint32_size * 2))

void *worker(void *arg) {
  struct ipc_command *command = (struct ipc_command *)arg;
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

void send_ipc_command(command_message cmdMsg, ipc_state_t *ipcState) {
  uint32_t write_offset = *ipcState->back_cb->write_offset;
  char *destination = ipcState->back_cb->memory_block + write_offset;
  memcpy(destination, &cmdMsg.command_type, uint32_size);
  // copy payload size as the same size as cmd type as they both uint_32t
  memcpy(destination + uint32_size, &cmdMsg.payload_size, uint32_size);
}

int check_rw_status(ipc_state_t *ipc_state) {
  int rdwr_offset = ipc_state->back_cb->rdwr_status;
  if (ipc_state->back_cb->memory_block[rdwr_offset] == READY_RDWR)
    return 0;
  if (ipc_state->back_cb->memory_block[rdwr_offset] == ACTIVE_RDWR)
    return -1;
}

void proccess_message_queue(data_context_t *data_context,
                            message_queue_t *message_queue,
                            thread_pool_t *tpool) {
  for (int i = message_queue->head; i < message_queue->tail; i++) {
    command_message cmd = message_queue->buffer[i];
    char *buffer = malloc(sizeof(char) * cmd.payload_size + 1);
    memcpy(buffer, cmd.payload, cmd.payload_size);
    buffer[cmd.payload_size] = '\0';
    command_handler_t *handler =
        get_command_handler(data_context, cmd.command_type, buffer);
    tpool_add_work(tpool, handler->func, handler);
    memset(&message_queue->buffer[i], 0, sizeof(command_message));
  }
  message_queue->head = 0;
  message_queue->count = 0;
  message_queue->tail = 0;
}

int enqueue_message(command_message cmd, message_queue_t *queue) {
  queue->buffer[queue->tail] = cmd;
  queue->tail = (queue->tail + 1) % message_queue_capacity;
  queue->count++;

  if (queue->tail > message_queue_capacity) {
    u_logger_warn("queue is overload\n");
    abort();
  }
}

void listen(void *ipc_state_arg) {
  u_logger_info("started listener...");
  ipc_state_t *ipc_state = (ipc_state_t *)ipc_state_arg;
  while (1) {
    char *memory = ipc_state->front_cb->memory_block;
    uint32_t *offset = ipc_state->front_cb->write_offset;
    command_message cmd = {0};

    char *start_ptr = memory + *offset;
    uint32_t message_type = *(uint32_t *)start_ptr;
    if (message_type == 0) {
      continue;
    }
    uint32_t message_payload_size = *(uint32_t *)(start_ptr + uint32_size);
    char *ptr_to_payload = start_ptr + (uint32_size * 2);
    ptr_to_payload[message_payload_size] = '\0';
    cmd.command_type = message_type;
    cmd.payload_size = message_payload_size;
    cmd.payload = ptr_to_payload;
    SHIFT_OFFSET(offset, message_payload_size);
    enqueue_message(cmd, ipc_state->message_queue);
  }
}

void *get_addresses(void *arg) {
  ipc_get_addresses_command *cmd = (ipc_get_addresses_command *)arg;
  // copy_to_buff(cmd->memory_ptr, cmd->buffer);
  return NULL;
}

void start_listener(ipc_state_t *ipc_state, thread_pool_t *tpool) {
  tpool_add_work(tpool, listen, ipc_state);
}

message_queue_t *init_message_queue() {
  message_queue_t *message_queue = malloc(sizeof(message_queue_t));
  message_queue->buffer =
      malloc(sizeof(command_message) * message_queue_capacity);
  message_queue->capacity = message_queue_capacity;
  message_queue->count = 0;
  message_queue->head = 0;
  message_queue->tail = 0;
  return message_queue;
}

ipc_state_t *initialize_shared_memory() {
  int fd = shm_open(FILE_NAME, O_CREAT | O_RDWR, 0666);
  if (fd == -1) {
    u_logger_error("error shm_open");
    return NULL;
  }

  ftruncate(fd, ADRESS_SPACE_SIZE);

  if (fd == -1) {
    u_logger_error("failed to create shared memory");
    return NULL;
  };
  u_logger_info("created shared memory");

  void *addr =
      mmap(NULL, ADRESS_SPACE_SIZE, PROT_READ | PROT_WRITE, MAP_SHARED, fd, 0);
  if (addr == NULL) {
    return NULL;
  }

  char *memory_block = (char *)addr;

  ipc_state_t *ipc = malloc(sizeof(ipc_state_t));

  control_block *back_cb = malloc(sizeof(control_block));
  control_block *front_cb = malloc(sizeof(control_block));

  back_cb->memory_block = memory_block + Bblock_addr_space;
  front_cb->memory_block = memory_block + Fblock_addr_space;

  int offset_start = 10;

  uint32_t *ptr_start_read_offset_f = (uint32_t *)(front_cb->memory_block);
  uint32_t *ptr_start_write_offset_f = (uint32_t *)front_cb->memory_block + 1;

  *ptr_start_write_offset_f = offset_start;
  *ptr_start_read_offset_f = offset_start;

  uint32_t *ptr_start_read_offset_b = (uint32_t *)(back_cb->memory_block);
  uint32_t *ptr_start_write_offset_b = (uint32_t *)back_cb->memory_block + 1;

  *ptr_start_write_offset_b = offset_start;
  *ptr_start_read_offset_b = offset_start;

  back_cb->read_offset = ptr_start_read_offset_b;
  back_cb->write_offset = ptr_start_write_offset_b;

  front_cb->read_offset = ptr_start_read_offset_f;
  front_cb->write_offset = ptr_start_write_offset_f;

  ipc->back_cb = back_cb;
  ipc->front_cb = front_cb;
  ipc->memory = memory_block;

  ipc->message_queue = init_message_queue();
  return ipc;
}

int copy_addrs_to_buffer(char *buffer, char **result_buffer,
                         int res_buffer_size, const char *delimiter) {
  int num_size = 0;
  char *str = strtok(buffer, delimiter);
  while (str != NULL) {
    ip_addr addr = malloc(sizeof(char) * strlen(str) + 1);
    addr[strlen(str)] = '\0';
    if (addr == NULL) {
      u_logger_error("error malloc on addr");
    }

    u_logger_info("length of addr %d\n", strlen(str));
    strcpy(addr, str);

    str = strtok(NULL, delimiter);
    result_buffer[num_size] = addr;
    if (num_size >= res_buffer_size) {
      return num_size; // TODO: prevent race conditions
    }
    num_size++;
  }
  return num_size;
}

void processes_ip_addrs_handler(void *command_handler_arg) {
  command_handler_t *command_handler = command_handler_arg;
  data_context_t *data_context = command_handler->data_context_t;
  int result = reallocate_addr_buffer(data_context);
  if (result == -1) {
    u_logger_error("error reallocating buffer");
    abort();
  }
  u_logger_info("buffer from received command %s", command_handler->buffer);

  int addr_count =
      copy_addrs_to_buffer(command_handler->buffer, data_context->addrs_buffer,
                           data_context->addr_capacity, ",");
  addr_count++;

  free(command_handler->buffer);
  free(command_handler);
  data_context->addr_count = addr_count;
}

void process_identify_host_handler(void *command_handler_arg) {
  command_handler_t *command_handler = command_handler_arg;
  data_context_t *data_context = command_handler->data_context_t;
  char *buffer[1] = {};

  copy_addrs_to_buffer(command_handler->buffer, buffer, 1, ",");
  data_context->host_addr = buffer[0];
  u_logger_info("%s", data_context->host_addr);
}

command_handler_t *get_command_handler(data_context_t *data_context,
                                       int cmd_type, char *buffer) {
  command_handler_t *handler = malloc(sizeof(command_handler_t));
  handler->data_context_t = data_context;
  handler->buffer = buffer;
  switch (cmd_type) {
  case CMD_GET_IP_ADDRS:
    u_logger_info("CMD_GET_IP_ADDRS %d", cmd_type);
    handler->func = processes_ip_addrs_handler;
    break;
  case CMD_IDENTIFY_HOST:
    u_logger_info("CMD_IDENTIFY_HOST %d", cmd_type);
    handler->func = process_identify_host_handler;
  default:
    break;
  }
  return handler;
}