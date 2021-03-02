#pragma once

#ifdef METADATA_EXPORTS
#define METADATA_API __declspec(dllexport)
#else
#define METADATA_API __declspec(dllimport)
#endif

extern "C" METADATA_API int retrieve_metadata(const WCHAR* filename, WCHAR* comment, int buffer_size);
