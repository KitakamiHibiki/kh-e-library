package com.kitakami.hibiki.elibrary.util

import android.content.Context
import android.content.Intent
import androidx.core.content.FileProvider
import java.io.File

/** Builds a chooser intent that shares [file] with other apps. */
fun shareFileIntent(context: Context, file: File): Intent {
    val uri = FileProvider.getUriForFile(
        context,
        "${context.packageName}.fileprovider",
        file,
    )
    val mime = when (file.extension.lowercase()) {
        "pdf" -> "application/pdf"
        else -> "application/epub+zip"
    }
    return Intent.createChooser(
        Intent(Intent.ACTION_SEND).apply {
            type = mime
            putExtra(Intent.EXTRA_STREAM, uri)
            addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
        },
        file.name,
    )
}
