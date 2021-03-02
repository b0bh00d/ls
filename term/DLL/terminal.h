#pragma once

#ifdef TERMINAL_EXPORTS
#define TERMINAL_API __declspec(dllexport)
#else
#define TERMINAL_API __declspec(dllimport)
#endif

extern "C" TERMINAL_API int HasColorSupport();
extern "C" TERMINAL_API int IsColorSupportEnabled();
extern "C" TERMINAL_API int EnableColorSupport(int enable);
