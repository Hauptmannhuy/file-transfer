#include "ipc.h"
#include "data_context.h"
#include "dependencies/cjson/cJSON.h"
#include "logger.h"
#include "tpool.h"
#include "utils.h"
#include <errno.h>
#include <inttypes.h>
#include <raylib.h>
#include <stddef.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <sys/eventfd.h>
#include <time.h>
#include <unistd.h>

#define uint32_size sizeof(uint32_t)
#define SHIFT_OFFSET(offset_ptr, payload_size)                                 \
  ((*offset_ptr) += (payload_size) + (uint32_size * 2))

char *get_destination_pointer(ipc_state_t *ipcState) {
  uint32_t write_offset = *ipcState->server_buffer->write_offset;
  char *destination = ipcState->server_buffer->memory_block + write_offset;
  return destination;
}

enum json_fields_serialize_num { ip, response_status, local_net_addrs };

const char *json_fields[] = {"ip", "status_response", "local_net_addrs"};

const char *json_data_field = "data";
const char *json_err_field = "error";

void add_handshake_response_to_json(cJSON *object, int status_accept) {
  cJSON_bool status;
  if (status_accept == 1) {
    status = cJSON_True;
  } else {
    status = cJSON_False;
  }

  cJSON_AddBoolToObject(object, json_fields[response_status], status);
}

void add_ip_to_json(cJSON *object, char *ip_str) {
  cJSON_AddStringToObject(object, json_fields[ip], ip_str);
}

cJSON *make_request_p2p_json(char *ip_str) {
  cJSON *object = cJSON_CreateObject();
  add_ip_to_json(object, ip_str);
  return object;
}

cJSON *make_accept_p2p_json(int status_accept, char *ip_str) {
  cJSON *object = cJSON_CreateObject();
  add_ip_to_json(object, ip_str);
  add_handshake_response_to_json(object, status_accept);
  return object;
}

parsed_json_t *parse_raw_json_bytes(char *json_buffer) {
  cJSON *parent_json = cJSON_Parse(json_buffer);
  if (parent_json == NULL) {
    const char *error = cJSON_GetErrorPtr();
    u_logger_error("error parse json: %s", error);
    return NULL;
  }
  cJSON *data = cJSON_GetObjectItem(parent_json, json_data_field);
  if (data == NULL) {
    u_logger_error("error getting object in json with key", json_data_field);
    return NULL;
  }
  cJSON *err = cJSON_GetObjectItem(parent_json, json_err_field);
  parsed_json_t *json = malloc(sizeof(parsed_json_t));
  json->parent_json = parent_json;
  json->data = data;
  json->err = err;
  return json;
}

int free_json(parsed_json_t *json) {
  cJSON_free(json->parent_json);
  free(json);
  const char *error = cJSON_GetErrorPtr();
  if (error != NULL) {
    u_logger_error("error free json: %s", error);
    return -1;
  }
  return 0;
}

// responsibility for free on send_ipc_command()
command_message new_cmd_msg(uint32_t command_type, cJSON *payload,
                            char *error) {
  cJSON *json = cJSON_CreateObject();
  command_message cmd = {0};
  if (error == NULL) {
    cJSON_AddNullToObject(json, "error");
  } else {
    cJSON_AddStringToObject(json, "error", error);
  }
  if (payload != NULL) {

    cJSON_AddItemToObject(json, "data", payload);
    char *json_str_payload = cJSON_PrintUnformatted(json);

    u_logger_info("json %s", json_str_payload);
    cmd.command_type = command_type;
    size_t length = strlen(json_str_payload);
    char *cmd_msg_buffer = malloc(sizeof(char) * length + 1);

    if (cmd_msg_buffer == NULL) {
      return cmd;
    }

    memcpy(cmd_msg_buffer, json_str_payload, length + 1);
    cmd_msg_buffer[length] = '\0';
    cmd.json_payload = cmd_msg_buffer;
    cmd.payload_size = length;
    cJSON_Delete(json);
  }
  return cmd;
}

void send_ipc_command(command_message cmdMsg, ipc_state_t *ipcState) {
  char *destination = get_destination_pointer(ipcState);
  // we move destination pointer on 4 bytes ahead from previous address for each
  // message field
  memcpy(destination, &cmdMsg.command_type, uint32_size);
  memcpy(destination + uint32_size, &cmdMsg.payload_size, uint32_size);
  if (cmdMsg.json_payload != NULL) {
    void *a = memcpy(destination + uint32_size * 2, cmdMsg.json_payload,
                     cmdMsg.payload_size);
  }

  eventfd_write(ipcState->uiEventFd, 1);
  u_logger_info("sending message with %d command_type, %d payload_size",
                cmdMsg.command_type, cmdMsg.payload_size);
  u_logger_info("message payload \n %s", cmdMsg.json_payload);

  if (cmdMsg.json_payload != NULL) {
    cJSON_free(cmdMsg.json_payload);
  }
}

void request_p2p(ipc_state_t *ipc, char *peer_ip) {
  command_message cmd_message =
      new_cmd_msg(CMD_REQUEST_P2P, make_request_p2p_json(peer_ip), NULL);
  send_ipc_command(cmd_message, ipc);
}

void get_local_network_hosts(ipc_state_t *ipc) {
  command_message cmd_message = {.command_type = CMD_GET_IP_ADDRS,
                                 .payload_size = 0};
  send_ipc_command(cmd_message, ipc);
}

void proccess_message_queue(data_context_t *data_context,
                            message_queue_t *message_queue,
                            thread_pool_t *tpool) {
  for (int i = message_queue->head; i < message_queue->tail; i++) {
    command_message cmd = message_queue->buffer[i];
    char *buffer = malloc(sizeof(char) * cmd.payload_size + 1);
    memcpy(buffer, cmd.json_payload, cmd.payload_size);
    buffer[cmd.payload_size] = '\0';
    u_logger_info("received message %s", buffer);
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
    return -1;
  }
  return 0;
}

command_message decode_message(ipc_state_t *ipc_state) {
  command_message cmd = {0};
  uint32_t *offset = ipc_state->gui_buffer->write_offset;
  char *start_ptr = ipc_state->gui_buffer->memory_block + *offset;
  uint32_t message_type = *(uint32_t *)start_ptr;

  uint32_t message_payload_size = *(uint32_t *)(start_ptr + uint32_size);
  char *ptr_to_payload = start_ptr + (uint32_size * 2);
  ptr_to_payload[message_payload_size] = '\0';
  cmd.command_type = message_type;
  cmd.payload_size = message_payload_size;
  cmd.json_payload = ptr_to_payload;
  return cmd;
}

void listen(void *ipc_state_arg) {
  u_logger_info("started listener...");
  ipc_state_t *ipc_state = (ipc_state_t *)ipc_state_arg;
  while (1) {
    u_logger_info("START READING EVENT FD AND SLEEP");
    eventfd_t read_counter;
    int event = eventfd_read(ipc_state->serverEventFd, &read_counter);

    if (event == -1) {
      u_logger_error("eventfd_read failed: %s\n", strerror(errno));
    }

    u_logger_info("eventfd val return %d", read_counter);

    command_message cmd = decode_message(ipc_state);
    SHIFT_OFFSET(ipc_state->gui_buffer->write_offset, cmd.payload_size);
    enqueue_message(cmd, ipc_state->message_queue);
  }
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

ipc_state_t *allocate_ipc(void *shm_addr, int serverFd, int uiEventFd) {
  char *memory_block = (char *)shm_addr;

  ipc_state_t *ipc = malloc(sizeof(ipc_state_t));

  control_block *server_buffer = malloc(sizeof(control_block));
  control_block *gui_buffer = malloc(sizeof(control_block));

  server_buffer->memory_block = memory_block + Bblock_addr_space;
  gui_buffer->memory_block = memory_block + Fblock_addr_space;

  uint32_t *ptr_start_read_offset_f = (uint32_t *)(gui_buffer->memory_block);
  uint32_t *ptr_start_write_offset_f = (uint32_t *)gui_buffer->memory_block + 1;

  uint32_t *ptr_start_read_offset_b = (uint32_t *)(server_buffer->memory_block);
  uint32_t *ptr_start_write_offset_b =
      (uint32_t *)server_buffer->memory_block + 1;

  server_buffer->read_offset = ptr_start_read_offset_b;
  server_buffer->write_offset = ptr_start_write_offset_b;

  gui_buffer->read_offset = ptr_start_read_offset_f;
  gui_buffer->write_offset = ptr_start_write_offset_f;

  ipc->server_buffer = server_buffer;
  ipc->gui_buffer = gui_buffer;
  ipc->memory = memory_block;
  ipc->serverEventFd = serverFd;
  ipc->uiEventFd = uiEventFd;

  ipc->message_queue = init_message_queue();
  return ipc;
}

ipc_state_t *initialize_shared_memory(char *eventFds[]) {
  int serverFd = strtol(eventFds[0], NULL, 10);
  int uiFd = strtol(eventFds[1], NULL, 10);

  char path[100] = "\0";

  snprintf(path, sizeof(path), "/%s", FILE_NAME);
  u_logger_info(path);
  int fd = shm_open(path, O_RDWR, 0666);
  if (fd == -1) {
    u_logger_error("error shm_open code: %d, reason: %s", errno,
                   strerror(errno));
    return NULL;
  }

  // ftruncate(fd, ADRESS_SPACE_SIZE);

  if (fd == -1) {
    u_logger_error("failed to create shared memory");
    return NULL;
  };
  u_logger_info("created shared memory");

  void *shm_addr =
      mmap(NULL, ADRESS_SPACE_SIZE, PROT_READ | PROT_WRITE, MAP_SHARED, fd, 0);
  if (shm_addr == NULL) {
    return NULL;
  }

  return allocate_ipc(shm_addr, serverFd, uiFd);
}

int copy_addrs_to_buffer(char *buffer, char **result_buffer,
                         int res_buffer_size, const char *delimiter) {
  int num_size = 0;
  char *str = strtok(buffer, delimiter);
  while (str != NULL) {
    char *addr = malloc(sizeof(char) * strlen(str) + 1);
    addr[strlen(str)] = '\0';

    if (addr == NULL) {
      u_logger_error("error malloc on addr");
    }

    u_logger_info("length of addr %d\n", strlen(str));
    strcpy(addr, str);

    str = strtok(NULL, delimiter);
    if (num_size >= res_buffer_size) {
      return num_size;
    }
    result_buffer[num_size] = addr;
    num_size++;
  }
  return num_size;
}

void process_local_net_peers(void *command_handler_arg) {
  command_handler_t *command_handler = command_handler_arg;
  data_context_t *data_context = command_handler->data_context_t;
  parsed_json_t *json = parse_raw_json_bytes(command_handler->buffer);
  if (json == NULL) {
    goto free_memory;
  }

  cJSON *arr = cJSON_GetObjectItem(json->data, json_fields[local_net_addrs]);
  if (arr == NULL) {
    u_logger_error("!!!!");
  }
  int arr_size = cJSON_GetArraySize(arr);
  u_logger_info("arr size %d", arr_size);
  for (int i = 0; i < arr_size; i++) {
    cJSON *current = cJSON_GetArrayItem(arr, i);
    u_logger_info("current %s", current->valuestring);
    char *ip = strdup(current->valuestring);
    if (!ip) {
      goto free_memory;
    }
    if (!includes_peer_ip(data_context->local_peers_dynamic_array, ip)) {
      array_push(data_context->local_peers_dynamic_array,
                 init_conn_peer(0, 0, ip));
    }
  }

free_memory:

  if (json->parent_json)
    cJSON_free(json->parent_json);
  if (json)
    free(json);

  free(command_handler->buffer);
  free(command_handler);
}

void process_identify_host_handler(void *command_handler_arg) {
  command_handler_t *command_handler = command_handler_arg;
  data_context_t *data_context = command_handler->data_context_t;
  char *buffer[1] = {};

  copy_addrs_to_buffer(command_handler->buffer, buffer, 1, ",");
  data_context->host_addr = buffer[0];
  u_logger_info("%s", data_context->host_addr);
}

void accept_p2p(ipc_state_t *ipc, char *peer_ip, data_context_t *data_context,
                int accept_status) {
  char *buffer = malloc(1);
  buffer[0] = accept_status;
  command_message msg = new_cmd_msg(
      CMD_ACCEPT_P2P, make_accept_p2p_json(accept_status, peer_ip), NULL);
  send_ipc_command(msg, ipc);
}

void process_accept_p2p(void *command_handler_arg) {
  command_handler_t *command_handler = command_handler_arg;
  data_context_t *data_context = command_handler->data_context_t;
  char *buffer[1] = {};

  copy_addrs_to_buffer(command_handler->buffer, buffer, 1, ",");
  // add_local_addr(data_context, buffer[0], 1);
}
command_handler_t *get_command_handler(data_context_t *data_context,
                                       int cmd_type, char *buffer) {
  command_handler_t *handler = malloc(sizeof(command_handler_t));
  handler->data_context_t = data_context;
  handler->buffer = buffer;
  switch (cmd_type) {
  case CMD_GET_IP_ADDRS:
    u_logger_info("CMD_GET_IP_ADDRS %d", cmd_type);
    handler->func = process_local_net_peers;
    break;
  case CMD_IDENTIFY_HOST:
    u_logger_info("CMD_IDENTIFY_HOST %d", cmd_type);
    handler->func = process_identify_host_handler;
    break;
  case CMD_ACCEPT_P2P:
    u_logger_info("CMD_ACCEPT_P2P %d", cmd_type);
    handler->func = process_accept_p2p;
    break;
  default:
    break;
  }
  return handler;
}
