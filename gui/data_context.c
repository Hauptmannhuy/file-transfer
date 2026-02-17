#include "data_context.h"
#include "logger.h"
#include <stdlib.h>
#define INITIAL_ADDRS_CAPACITY 500

data_context_t *data_context_init() {
  data_context_t *context = malloc(sizeof(data_context_t));
  context->addr_capacity = INITIAL_ADDRS_CAPACITY;
  context->addr_count = 0;
  context->local_addrs_buffer =
      calloc(INITIAL_ADDRS_CAPACITY, sizeof(conn_peer_t));
  context->host_addr = NULL;
  return context;
}

int reallocate_local_addr_buffer(data_context_t *data_context) {
  if (data_context->local_addrs_buffer == NULL) {
    u_logger_error("addrs buffer is null");
    return -1;
  }

  for (int i = 0; i < data_context->addr_capacity; i++) {
    free(data_context->local_addrs_buffer[i]);
  }

  void *buffer_ptr = calloc(data_context->addr_capacity, sizeof(conn_peer_t));
  if (buffer_ptr == NULL) {
    u_logger_error("error reallocating addrs buffer");
    return -1;
  }

  data_context->local_addrs_buffer = buffer_ptr;

  return 0;
}

int add_local_addr(data_context_t *data_context, char *ip, int pending) {
  u_logger_info("adding local pending peer");
  conn_peer_t *peer = init_conn_peer(0, pending, ip);
  peer->pending = pending;
  data_context->local_addrs_buffer[data_context->addr_count] = peer;
  data_context->addr_count+=1;
  return 0;
}

conn_peer_t *init_conn_peer(int connected, int pending, char *ip) {
  conn_peer_t *peer = malloc(sizeof(conn_peer_t));
  peer->connected = connected;
  peer->pending = pending;
  peer->ip = ip;
  return peer;
}

int is_connection_established(data_context_t *data_context, char *ip) {
  for (int i = 0; i < data_context->addr_count; i++) {
    conn_peer_t *peer = data_context->local_addrs_buffer[i];
    int equal = strcmp(ip, peer->ip);
    if (equal == 0) {
      return peer->connected;
    }
  }
  return -1;
}