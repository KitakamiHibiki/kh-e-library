package com.kitakami.hibiki.elibrary.ui.bookdetail

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.SuggestionChip
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import coil.compose.AsyncImage
import coil.request.ImageRequest
import com.kitakami.hibiki.elibrary.AppContainer
import com.kitakami.hibiki.elibrary.KHApplication
import com.kitakami.hibiki.elibrary.R
import com.kitakami.hibiki.elibrary.data.model.Book
import com.kitakami.hibiki.elibrary.ui.bookshelf.coverUrlFor
import com.kitakami.hibiki.elibrary.ui.bookshelf.components.ShelfTagDialog
import com.kitakami.hibiki.elibrary.ui.theme.BrandBlue
import com.kitakami.hibiki.elibrary.ui.theme.BrandRed
import com.kitakami.hibiki.elibrary.util.shareFileIntent
import kotlinx.coroutines.launch
import java.io.File

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun BookDetailScreen(
    container: AppContainer,
    bookId: Long,
    onBack: () -> Unit,
    onRead: (Long) -> Unit,
) {
    val viewModel: BookDetailViewModel = viewModel {
        val app = this[ViewModelProvider.AndroidViewModelFactory.APPLICATION_KEY] as KHApplication
        BookDetailViewModel(app.container)
    }
    val baseUrl by container.baseUrl.collectAsStateWithLifecycle()
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val snackbarHostState = remember { SnackbarHostState() }

    var menuExpanded by remember { mutableStateOf(false) }
    var shelfDialog by remember { mutableStateOf(false) }
    var deleteConfirm by remember { mutableStateOf(false) }
    var exporting by remember { mutableStateOf(false) }

    LaunchedEffect(bookId) { viewModel.load(bookId) }

    // Navigate away once deleted.
    LaunchedEffect(viewModel.deleted) {
        if (viewModel.deleted) onBack()
    }

    LaunchedEffect(viewModel.message) {
        val msg = viewModel.message ?: return@LaunchedEffect
        snackbarHostState.showSnackbar(context.getString(msg.resId, *msg.args.toTypedArray()))
        viewModel.consumeMessage()
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(stringResource(R.string.book_detail_title)) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.reader_back))
                    }
                },
                actions = {
                    IconButton(onClick = { menuExpanded = true }) {
                        Icon(Icons.Default.MoreVert, contentDescription = null)
                    }
                    DropdownMenu(expanded = menuExpanded, onDismissRequest = { menuExpanded = false }) {
                        val status = viewModel.book?.read_status
                        if (status != "reading") {
                            DropdownMenuItem(
                                text = { Text(stringResource(R.string.menu_mark_reading)) },
                                onClick = { menuExpanded = false; viewModel.setReadStatus("reading") },
                            )
                        }
                        if (status != "finished") {
                            DropdownMenuItem(
                                text = { Text(stringResource(R.string.menu_mark_finished)) },
                                onClick = { menuExpanded = false; viewModel.setReadStatus("finished") },
                            )
                        }
                        if (status != "unread") {
                            DropdownMenuItem(
                                text = { Text(stringResource(R.string.menu_mark_unread)) },
                                onClick = { menuExpanded = false; viewModel.setReadStatus("unread") },
                            )
                        }
                        DropdownMenuItem(
                            text = { Text(stringResource(R.string.menu_add_to_shelf)) },
                            onClick = { menuExpanded = false; shelfDialog = true },
                        )
                        DropdownMenuItem(
                            text = { Text(stringResource(R.string.menu_reprocess)) },
                            onClick = { menuExpanded = false; viewModel.reprocess() },
                        )
                        DropdownMenuItem(
                            text = { Text(stringResource(R.string.menu_export)) },
                            onClick = {
                                menuExpanded = false
                                val book = viewModel.book ?: return@DropdownMenuItem
                                scope.launch {
                                    exporting = true
                                    try {
                                        val ext = if (book.isEpub) "epub" else "pdf"
                                        val file = File(
                                            context.cacheDir,
                                            "export_${book.id}_${book.title.replace(Regex("[\\\\/:*?\"<>|]"), "_")}.$ext",
                                        )
                                        container.repository.downloadBookToFile(book.id, file)
                                        context.startActivity(shareFileIntent(context, file))
                                    } catch (e: Exception) {
                                        snackbarHostState.showSnackbar(context.getString(R.string.bookshelf_load_failed, e.message ?: ""))
                                    } finally {
                                        exporting = false
                                    }
                                }
                            },
                        )
                        DropdownMenuItem(
                            text = { Text(stringResource(R.string.menu_delete), color = BrandRed) },
                            onClick = { menuExpanded = false; deleteConfirm = true },
                        )
                    }
                },
            )
        },
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { innerPadding ->
        val book = viewModel.book
        when {
            viewModel.loading && book == null -> {
                Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) {
                    CircularProgressIndicator()
                }
            }
            book == null -> {
                Column(
                    modifier = Modifier.fillMaxSize().padding(innerPadding),
                    verticalArrangement = Arrangement.Center,
                    horizontalAlignment = Alignment.CenterHorizontally,
                ) {
                    Text(viewModel.error ?: "", color = MaterialTheme.colorScheme.error)
                    Button(onClick = { viewModel.load(bookId) }, modifier = Modifier.padding(top = 12.dp)) {
                        Text(stringResource(R.string.bookshelf_retry))
                    }
                }
            }
            else -> {
                BookDetailContent(
                    book = book,
                    coverUrl = coverUrlFor(baseUrl, book.id),
                    onRead = { onRead(book.id) },
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(innerPadding),
                )
            }
        }
    }

    // Dialogs.
    viewModel.book?.let { book ->
        if (shelfDialog) {
            ShelfTagDialog(
                book = book,
                tags = viewModel.tags,
                onDismiss = { shelfDialog = false },
                onSave = { addIds, removeIds ->
                    viewModel.updateBookTags(addIds, removeIds)
                    shelfDialog = false
                },
            )
        }
        if (deleteConfirm) {
            AlertDialog(
                onDismissRequest = { deleteConfirm = false },
                title = { Text(stringResource(R.string.menu_delete)) },
                text = { Text(stringResource(R.string.menu_delete_confirm, book.title)) },
                confirmButton = {
                    TextButton(onClick = {
                        deleteConfirm = false
                        viewModel.delete()
                    }) {
                        Text(stringResource(R.string.common_delete), color = BrandRed)
                    }
                },
                dismissButton = {
                    TextButton(onClick = { deleteConfirm = false }) {
                        Text(stringResource(R.string.common_cancel))
                    }
                },
            )
        }
        if (exporting) {
            AlertDialog(
                onDismissRequest = {},
                title = { Text(stringResource(R.string.menu_export)) },
                text = {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        CircularProgressIndicator(Modifier.width(24.dp).height(24.dp), strokeWidth = 2.dp)
                        Spacer(Modifier.width(12.dp))
                        Text(stringResource(R.string.common_loading))
                    }
                },
                confirmButton = {},
            )
        }
    }
}

@Composable
private fun BookDetailContent(
    book: Book,
    coverUrl: String,
    onRead: () -> Unit,
    modifier: Modifier = Modifier,
) {
    Column(
        modifier = modifier
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
    ) {
        Row(modifier = Modifier.fillMaxWidth()) {
            CoverThumb(book = book, coverUrl = coverUrl)
            Spacer(Modifier.width(16.dp))
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = book.title,
                    style = MaterialTheme.typography.headlineSmall,
                )
                Spacer(Modifier.height(4.dp))
                Text(
                    text = book.author.ifBlank { stringResource(R.string.book_unknown_author) },
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                if (book.tags.isNotEmpty()) {
                    Spacer(Modifier.height(8.dp))
                    Row(horizontalArrangement = Arrangement.spacedBy(4.dp)) {
                        book.tags.take(4).forEach { tag ->
                            SuggestionChip(onClick = {}, label = { Text(tag) })
                        }
                    }
                }
            }
        }

        Spacer(Modifier.height(20.dp))

        InfoRow(stringResource(R.string.book_status), statusLabelText(book.read_status))
        InfoRow(stringResource(R.string.book_format), book.file_type.uppercase())
        InfoRow(stringResource(R.string.book_size), formatFileSize(book.file_size))
        InfoRow(
            stringResource(if (book.isEpub) R.string.book_pages_epub else R.string.book_pages_pdf),
            if (book.pages > 0) book.pages.toString() else "-",
        )
        book.isbn?.takeIf { it.isNotBlank() }?.let { InfoRow(stringResource(R.string.book_isbn), it) }
        book.publisher?.takeIf { it.isNotBlank() }?.let { InfoRow(stringResource(R.string.book_publisher), it) }
        book.language?.takeIf { it.isNotBlank() }?.let { InfoRow(stringResource(R.string.book_language), it) }

        val description = book.description?.takeIf { it.isNotBlank() }
        if (description != null) {
            Spacer(Modifier.height(16.dp))
            Text(
                text = stringResource(R.string.book_description),
                style = MaterialTheme.typography.titleSmall,
            )
            Spacer(Modifier.height(4.dp))
            Text(
                text = description,
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }

        Spacer(Modifier.height(24.dp))
        Button(
            onClick = onRead,
            modifier = Modifier.fillMaxWidth(),
        ) {
            Text(stringResource(R.string.book_read))
        }
    }
}

@Composable
private fun CoverThumb(book: Book, coverUrl: String) {
    val shape = RoundedCornerShape(8.dp)
    Box(
        modifier = Modifier
            .width(120.dp)
            .height(160.dp)
            .clip(shape)
            .background(MaterialTheme.colorScheme.surfaceVariant),
    ) {
        if (book.cover.isNotEmpty()) {
            AsyncImage(
                model = ImageRequest.Builder(LocalContext.current)
                    .data(coverUrl)
                    .crossfade(true)
                    .build(),
                contentDescription = book.title,
                contentScale = ContentScale.Crop,
                modifier = Modifier.fillMaxSize(),
            )
        } else {
            Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                Text(
                    text = if (book.isEpub) "EPUB" else "PDF",
                    style = MaterialTheme.typography.titleMedium,
                    color = if (book.isEpub) BrandBlue else BrandRed,
                )
            }
        }
    }
}

@Composable
private fun InfoRow(label: String, value: String) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 4.dp),
        verticalAlignment = Alignment.Top,
    ) {
        Text(
            text = label,
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.width(88.dp),
        )
        Text(
            text = value,
            style = MaterialTheme.typography.bodyMedium,
            maxLines = 2,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.weight(1f),
        )
    }
}

@Composable
private fun statusLabelText(status: String): String = stringResource(
    when (status) {
        "reading" -> R.string.status_reading
        "finished" -> R.string.status_finished
        else -> R.string.status_unread
    },
)

private fun formatFileSize(bytes: Long): String {
    if (bytes <= 0) return "-"
    return when {
        bytes < 1024 -> "$bytes B"
        bytes < 1024 * 1024 -> "%.1f KB".format(bytes / 1024.0)
        bytes < 1024L * 1024 * 1024 -> "%.1f MB".format(bytes / 1024.0 / 1024.0)
        else -> "%.2f GB".format(bytes / 1024.0 / 1024.0 / 1024.0)
    }
}
