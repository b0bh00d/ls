#include "pch.h"

// metadata.cpp : This file contains the 'main' function. Program execution begins
// and ends there.
//
// https://stackoverflow.com/questions/1713214/how-to-use-c-in-go
// https://github.com/arrieta/golang-cpp-basic-example

#include <locale>
#include <codecvt>
#include <iostream>
#include <sstream>
#include <string>

#include "metadata.h"

// this method will extract and return any Directory Opus metadata (i.e.,
// description) that it finds in the specified directory/file.
bool _get_ads(const WCHAR* filename, WCHAR* comment, int& buffer_size)
{
	bool result{false};
	std::wstring wfilename(filename);
	buffer_size /= 2; // convert to UTF-16

	//using convert_type = std::codecvt_utf8<wchar_t>;
	//std::wstring_convert<convert_type, wchar_t> converter;
	////use converter (.to_bytes: wstr->str, .from_bytes: str->wstr)
	//std::wstring wfilename = converter.from_bytes(filename);

	auto attr = GetFileAttributes(filename);
	if (attr & FILE_ATTRIBUTE_DIRECTORY)
	{
		std::wstringstream ss;
		ss << wfilename << L":\007OpusMetaInformation";
		auto wmetaname = ss.str();

		auto hFile = ::CreateFile(wmetaname.c_str(), GENERIC_READ, 0, NULL, OPEN_EXISTING, FILE_ATTRIBUTE_NORMAL, 0);
		if (hFile != INVALID_HANDLE_VALUE)
		{
			UINT8 inBuffer[1024];
			DWORD nBytesRead{0};
			auto hr = ::ReadFile(hFile, &inBuffer, 16, &nBytesRead, nullptr);

			UINT32* p = reinterpret_cast<UINT32*>(&inBuffer[0]);
			auto size = *p++;
			auto flags = *p++;
			auto rating = *p++;
			auto comment_size = *p++;

			size -= 16;
			// skip any padding
			hr = ::ReadFile(hFile, &inBuffer, size, &nBytesRead, nullptr);
			hr = ::ReadFile(hFile, &inBuffer, (comment_size + 1) * 2, &nBytesRead, nullptr);

			auto wcomment = std::wstring(reinterpret_cast<LPTSTR>(&inBuffer[0]));

			//using convert_type = std::codecvt_utf8<wchar_t>;
			//std::wstring_convert<convert_type, wchar_t> converter;
			////use converter (.to_bytes: wstr->str, .from_bytes: str->wstr)
			//std::string mbcomment = converter.to_bytes(wcomment);

			::wcscpy_s(comment, buffer_size, wcomment.c_str());
			// tell the caller how many WCHARs are valid in the buffer
			buffer_size = static_cast<int>(wcomment.length());
			result = true;

			CloseHandle(hFile);
		}
	}
	else // https://www.codeproject.com/articles/16314/access-the-summary-information-property-set-of-a-f
	{
		IPropertySetStorage* pPropSetStg{nullptr};
		auto m = STGM_SHARE_EXCLUSIVE | STGM_READ;
		auto hr = ::StgOpenStorageEx(
			wfilename.c_str(), m, STGFMT_FILE, 0, nullptr, nullptr, IID_IPropertySetStorage, reinterpret_cast<void**>(&pPropSetStg));

		if (!FAILED(hr))
		{
			IPropertyStorage* pSet;
			PROPSPEC propspec; // PROPSPEC rgpspec[]{PIDSI_COMMENTS};
			PROPVARIANT propRead;

			propspec.ulKind = PRSPEC_PROPID;
			propspec.propid = PIDSI_COMMENTS;

			hr = pPropSetStg->Open(FMTID_SummaryInformation, m, &pSet);
			if (!FAILED(hr))
			{
				hr = pSet->ReadMultiple(1, &propspec, &propRead);
				if (!FAILED(hr))
				{
					auto wcomment = std::wstring(reinterpret_cast<LPTSTR>(propRead.pwszVal));

					//using convert_type = std::codecvt_utf8<wchar_t>;
					//std::wstring_convert<convert_type, wchar_t> converter;
					////use converter (.to_bytes: wstr->str, .from_bytes: str->wstr)
					//std::string mbcomment = converter.to_bytes(wcomment);

					::wcscpy_s(comment, buffer_size, wcomment.c_str());
					// tell the caller how many WCHARs are valid in the buffer
					buffer_size = static_cast<int>(wcomment.length());
					result = true;
				}
			}
		}
	}

	return result;
}

int retrieve_metadata(const WCHAR* filename, WCHAR* comment, int buffer_size)
{
	int result = _get_ads(filename, comment, buffer_size) ? 1 : 0;
	return result ? buffer_size : 0;
}
