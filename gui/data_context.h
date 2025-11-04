#pragma once

typedef char *ip_addr;

typedef struct data_context_t {
  ip_addr *addrs_buffer;
  ip_addr host_addr;
  int addr_capacity;
  int addr_count;
} data_context_t;

int reallocate_addr_buffer(data_context_t *data_context);
data_context_t *data_context_init();
