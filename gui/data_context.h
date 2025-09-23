#pragma once

typedef char *ip_addr;

typedef struct data_context_t {
  ip_addr *addrs_buffer;
  int addr_capacity;
  int addr_count;
} data_context_t;

data_context_t *data_context_init();
