#include "utils.h"
#include "data_context.h"
#include <stdio.h>
#include <stdlib.h>
#define INIT_CAPACITY 256

int *new_int_array() {
  Array_header_t *header =
      malloc(sizeof(int) * INIT_CAPACITY + sizeof(Array_header_t));

  header->capacity = INIT_CAPACITY;
  header->count = 0;

  int *numbers = (int *)(header + 1);
  return numbers;
}

Array_header_t *get_header(void *array) {
  Array_header_t *header = ((Array_header_t *)array) - 1;
  return header;
}

void *array_init_impl(size_t elem_size) {
  Array_header_t *header = (Array_header_t *)malloc(sizeof(Array_header_t) +
                                                    elem_size * init_capacity);

  if (!header)
    return NULL;

  header->count = 0;
  header->capacity = init_capacity;

  return (void *)(header + 1);
}

int free_dynamic_array(void *array) {
  Array_header_t *header = get_header(array);
}

void print_dynamic_array_int(int *array) {
  Array_header_t *header = ((Array_header_t *)array) - 1;
  for (int i = 0; i < header->count; i++) {
    printf("%d\n", array[i]);
  }
}

void print_dynamic_array_float(float *array) {
  Array_header_t *header = ((Array_header_t *)array) - 1;
  for (int i = 0; i < header->count; i++) {
    printf("%f\n", array[i]);
  }
}

void print_dynamic_array_str(char *array) {
  Array_header_t *header = ((Array_header_t *)array) - 1;
  for (int i = 0; i < header->count; i++) {
    printf("%c", array[i]);
  }
  printf("\n");
}

void print_dynamic_array_peer(conn_peer_t **array) {
  Array_header_t *header = ((Array_header_t *)array) - 1;
  for (int i = 0; i < header->count; i++) {
    printf("peer ip %s\n", array[i]->ip);
  }
}