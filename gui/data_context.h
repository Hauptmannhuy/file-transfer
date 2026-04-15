#pragma once

typedef struct conn_peer_t {
  char *ip;
  int connected;
  int pending;
} conn_peer_t;

typedef struct data_context_t {
  conn_peer_t **local_peers_dynamic_array;
  char *host_addr;
} data_context_t;

int is_connection_established(data_context_t *data_context, char *ip);
int is_connection_pending(data_context_t *data_context, char *ip);
bool includes_peer_ip(conn_peer_t **dynamic_array, char *ip);

conn_peer_t *init_conn_peer(int connected, int pending, char *ip);
data_context_t *data_context_init();
