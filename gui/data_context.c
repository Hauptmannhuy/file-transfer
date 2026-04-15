#include "data_context.h"
#include "logger.h"
#include "utils.h"
#include <stdbool.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>

data_context_t *data_context_init() {
  data_context_t *context = malloc(sizeof(data_context_t));
  context->local_peers_dynamic_array = array_init(conn_peer_t *);
  context->host_addr = NULL;
  return context;
}

conn_peer_t *init_conn_peer(int connected, int pending, char *ip) {
  conn_peer_t *peer = malloc(sizeof(conn_peer_t));
  peer->connected = connected;
  peer->pending = pending;
  peer->ip = ip;
  return peer;
}

bool includes_peer_ip(conn_peer_t **dynamic_array, char *ip) {
  Array_header_t *header = get_header(dynamic_array);

  for (int i = 0; i < header->count; i++) {
    if (strcmp(dynamic_array[i]->ip, ip) == 0) {
      u_logger_info("%s == %s", dynamic_array[i]->ip, ip);
      return true;
    }
  }
  return false;
}

int is_connection_established(data_context_t *data_context, char *ip) {
  Array_header_t *header = get_header(data_context->local_peers_dynamic_array);

  for (int i = 0; i < header->count; i++) {
    conn_peer_t *peer = data_context->local_peers_dynamic_array[i];
    int equal = strcmp(ip, peer->ip);
    if (equal == 0) {
      return peer->connected;
    }
  }
  return -1;
}

int is_connection_pending(data_context_t *data_context, char *ip) {
  Array_header_t *header = get_header(data_context->local_peers_dynamic_array);

  for (int i = 0; i < header->count; i++) {
    conn_peer_t *peer = data_context->local_peers_dynamic_array[i];
    int equal = strcmp(ip, peer->ip);
    if (equal == 0) {
      return peer->pending;
    }
  }
  return -1;
}
