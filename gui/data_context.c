#include "data_context.h"
#include <stdlib.h>

#define INITIAL_ADDRS_CAPACITY 5

data_context_t *data_context_init() {
  data_context_t *context = malloc(sizeof(data_context_t));
  context->addr_capacity = INITIAL_ADDRS_CAPACITY;
  context->addr_count = 0;
  context->addrs_buffer = calloc(INITIAL_ADDRS_CAPACITY, sizeof(ip_addr));
  return context;
}