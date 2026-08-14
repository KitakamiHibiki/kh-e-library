package com.kitakami.hibiki.elibrary.ui.reader

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.kitakami.hibiki.elibrary.AppContainer
import com.kitakami.hibiki.elibrary.R
import com.kitakami.hibiki.elibrary.data.model.Book
import com.kitakami.hibiki.elibrary.data.remote.toApiMessage

/** Chooses the EPUB or PDF reader based on the book's file type. */
@Composable
fun ReaderHostScreen(
    container: AppContainer,
    bookId: Long,
    onBack: () -> Unit,
    onFinished: () -> Unit,
) {
    var book by remember { mutableStateOf<Book?>(null) }
    var error by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(bookId) {
        try {
            book = container.repository.getBook(bookId)
        } catch (e: Exception) {
            error = e.toApiMessage()
        }
    }

    when {
        book == null && error == null -> {
            Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                CircularProgressIndicator()
            }
        }
        error != null -> {
            Column(
                modifier = Modifier.fillMaxSize(),
                verticalArrangement = Arrangement.Center,
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                Text(
                    stringResource(R.string.reader_open_failed, error ?: ""),
                    color = MaterialTheme.colorScheme.error,
                    modifier = Modifier.padding(24.dp),
                )
                Button(onClick = onBack, modifier = Modifier.padding(top = 12.dp)) {
                    Text(stringResource(R.string.reader_back))
                }
            }
        }
        book != null && book!!.isEpub -> {
            EpubReaderScreen(container = container, bookId = bookId, onBack = onFinished)
        }
        else -> {
            PdfReaderScreen(container = container, bookId = bookId, onBack = onFinished)
        }
    }
}
