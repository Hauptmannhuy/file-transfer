#include "data_context.h"
#include "logger.h"
#include <stdlib.h>
#define INITIAL_ADDRS_CAPACITY 5

data_context_t *data_context_init() {
  data_context_t *context = malloc(sizeof(data_context_t));
  context->addr_capacity = INITIAL_ADDRS_CAPACITY;
  context->addr_count = 0;
  context->addrs_buffer = calloc(INITIAL_ADDRS_CAPACITY, sizeof(ip_addr));
  return context;
}

int reallocate_addr_buffer(data_context_t *data_context) {
  if (data_context->addrs_buffer == NULL) {
    u_logger_error("addrs buffer is null");
    return -1;
  }

  for (int i = 0; i < INITIAL_ADDRS_CAPACITY; i++) {
    free(data_context->addrs_buffer[i]);
  }

  if (data_context->addrs_buffer != NULL) {
    free(data_context->addrs_buffer);
  }

  void *buffer_ptr = calloc(INITIAL_ADDRS_CAPACITY, sizeof(ip_addr));
  if (buffer_ptr == NULL) {
    u_logger_error("error reallocating addrs buffer");
    return -1;
  }

  data_context->addrs_buffer = buffer_ptr;

  return 0;
}