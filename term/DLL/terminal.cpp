#include "pch.h"

#include <locale>
#include <codecvt>
#include <iostream>
#include <sstream>
#include <string>

#include "terminal.h"

#pragma comment(lib, "Shlwapi.lib")

// https://gist.github.com/mlocati/21a9233ac83f7d3d7837535bc109b3b7
#ifdef _WIN32
#ifndef ENABLE_VIRTUAL_TERMINAL_PROCESSING
#define ENABLE_VIRTUAL_TERMINAL_PROCESSING 0x0004
#endif
#ifndef NTSTATUS
typedef long NTSTATUS;
#endif
typedef NTSTATUS(WINAPI* RtlGetVersionPtr)(PRTL_OSVERSIONINFOW);
#endif
/**
	* Check if the current console supports ANSI colors.
	*/
bool has_color_support()
{
	static BOOL result = 2;
	if (result == 2)
	{
		const DWORD MINV_MAJOR = 10, MINV_MINOR = 0, MINV_BUILD = 10586;
		result = FALSE;
		HMODULE hMod = GetModuleHandle(TEXT("ntdll.dll"));
		if (hMod)
		{
			RtlGetVersionPtr fn = (RtlGetVersionPtr)GetProcAddress(hMod, "RtlGetVersion");
			if (fn != NULL)
			{
				RTL_OSVERSIONINFOW rovi = {0};
				rovi.dwOSVersionInfoSize = sizeof(rovi);
				if (fn(&rovi) == 0)
				{
					if (rovi.dwMajorVersion > MINV_MAJOR ||
						(rovi.dwMajorVersion == MINV_MAJOR &&
						 (rovi.dwMinorVersion > MINV_MINOR || (rovi.dwMinorVersion == MINV_MINOR && rovi.dwBuildNumber >= MINV_BUILD))))
					{
						result = TRUE;
					}
				}
			}
		}
	}
	return result == TRUE;
}

int HasColorSupport()
{
	return has_color_support() ? 1 : 0;
}

/**
	* Check if the current console has ANSI colors enabled.
	*/
bool is_color_support_snabled()
{
	BOOL result = FALSE;
	if (has_color_support())
	{
		HANDLE hStdOut = GetStdHandle(STD_OUTPUT_HANDLE);
		if (hStdOut != INVALID_HANDLE_VALUE)
		{
			DWORD mode;
			if (GetConsoleMode(hStdOut, &mode))
			{
				if (mode & ENABLE_VIRTUAL_TERMINAL_PROCESSING)
				{
					result = TRUE;
				}
			}
		}
	}

	return result == TRUE;
}

int IsColorSupportEnabled()
{
	return is_color_support_snabled() ? 1 : 0;
}

/**
	* Enable/disable ANSI colors support for the current console.
	*
	* Returns TRUE if operation succeeded, FALSE otherwise.
	*/
bool enable_color_support(bool enabled)
{
	BOOL result = FALSE;
	if (has_color_support())
	{
		HANDLE hStdOut;
		hStdOut = GetStdHandle(STD_OUTPUT_HANDLE);
		if (hStdOut != INVALID_HANDLE_VALUE)
		{
			DWORD mode;
			if (GetConsoleMode(hStdOut, &mode))
			{
				if (((mode & ENABLE_VIRTUAL_TERMINAL_PROCESSING) ? 1 : 0) == (enabled ? 1 : 0))
				{
					result = TRUE;
				}
				else
				{
					if (enabled)
					{
						mode |= ENABLE_VIRTUAL_TERMINAL_PROCESSING;
					}
					else
					{
						mode &= ~ENABLE_VIRTUAL_TERMINAL_PROCESSING;
					}
					if (SetConsoleMode(hStdOut, mode))
					{
						result = TRUE;
					}
				}
			}
		}
	}

	return result == TRUE;
}

int EnableColorSupport(int enable)
{
	return enable_color_support(enable == 1 ? true : false) ? 1 : 0;
}
