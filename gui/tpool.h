#pragma once

#include <pthread.h>
#include <stdbool.h>
#include <stddef.h>

typedef void (*thread_func_t)(void *arg);

typedef struct thread_pool_work_t {
  struct thread_pool_work_t *next;
  thread_func_t func;
  void *arg;
} thread_pool_work_t;

typedef struct thread_pool_t {
  size_t working_cnt;
  size_t thread_cnt;
  thread_pool_work_t *work_first;
  thread_pool_work_t *work_last;
  pthread_mutex_t mutex;
  pthread_cond_t work_cond;
  pthread_cond_t working_cond;
  bool stop;

} thread_pool_t;

bool tpool_add_work(thread_pool_t *tm, thread_func_t func, void *arg);
void tpool_wait(thread_pool_t *tm);

thread_pool_t *create_tpool(int thread_num);
