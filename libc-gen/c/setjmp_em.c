#include <setjmp.h>
#include <stdint.h>

void __wasm_setjmp(void* env, uint32_t label, void* func_invocation_id) {
  struct jmp_buf_impl* jb = env;
  jb->func_invocation_id = func_invocation_id;
  jb->label = label;
}

uint32_t __wasm_setjmp_test(void* env, void* func_invocation_id) {
  struct jmp_buf_impl* jb = env;
  if (jb->func_invocation_id == func_invocation_id) return jb->label;
  return 0;
}

_Thread_local uintptr_t __THREW__ = 0;
_Thread_local int __threwValue = 0;
void _throw_longjmp();

void emscripten_longjmp(uintptr_t env, int val) {
  if (val == 0) val = 1;
  __threwValue = val;
  __THREW__ = env;
  _throw_longjmp();
}

_Thread_local static int tempRet0 = 0;
int getTempRet0() { return tempRet0; }
void setTempRet0(int value) { tempRet0 = value; }
