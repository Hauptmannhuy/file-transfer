#pragma once

typedef struct conn_peer_t {
  char *ip;
  int connected;
} conn_peer_t;

typedef struct data_context_t {
  conn_peer_t **addrs_buffer;
  char *host_addr;
  int addr_capacity;
  int addr_count;
} data_context_t;

int is_connection_established(data_context_t *data_context, char *ip);
int reallocate_addr_buffer(data_context_t *data_context);
conn_peer_t *init_conn_peer();
data_context_t *data_context_init();
