#pragma once

#include "stdlib.h"

typedef struct Array_header_t {
  int count;
  int capacity;
} Array_header_t;

static const int init_capacity = 1;

void *array_init_impl(size_t elem_size);

#define array_init(type) (type *)array_init_impl(sizeof(type));

#define array_push(arr, x)                                                     \
  do {                                                                         \
    if (arr == NULL) {                                                         \
      Array_header_t *header =                                                 \
          malloc(sizeof(Array_header_t) + sizeof(*arr) * init_capacity);       \
      header->count = 0;                                                       \
      header->capacity = init_capacity;                                        \
      arr = (void *)(header + 1);                                              \
    }                                                                          \
    Array_header_t *header = (Array_header_t *)arr - 1;                        \
    if (header->count >= header->capacity) {                                   \
      header->capacity *= 2;                                                   \
      header = realloc(header, (sizeof(*arr) * header->capacity) +             \
                                   sizeof(Array_header_t));                    \
      arr = (void *)(header + 1);                                              \
    }                                                                          \
    arr[header->count] = x;                                                    \
    header->count++;                                                           \
  } while (0);

int free_dynamic_array(void *array);
Array_header_t *get_header(void *array);