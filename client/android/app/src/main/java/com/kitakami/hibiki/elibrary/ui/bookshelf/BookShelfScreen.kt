package com.kitakami.hibiki.elibrary.ui.bookshelf

import android.content.Context
import android.net.Uri
import android.provider.OpenableColumns
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items as gridItems
import androidx.compose.foundation.lazy.grid.rememberLazyGridState
import androidx.compose.foundation.lazy.items as lazyItems
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Check
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.Menu
import androidx.compose.material.icons.filled.Search
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material.icons.filled.Sort
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.Checkbox
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DrawerValue
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalDrawerSheet
import androidx.compose.material3.ModalNavigationDrawer
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.rememberDrawerState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import com.kitakami.hibiki.elibrary.AppContainer
import com.kitakami.hibiki.elibrary.KHApplication
import com.kitakami.hibiki.elibrary.R
import com.kitakami.hibiki.elibrary.data.model.Book
import com.kitakami.hibiki.elibrary.data.model.Tag
import com.kitakami.hibiki.elibrary.ui.bookshelf.components.BookCard
import com.kitakami.hibiki.elibrary.ui.bookshelf.components.BookCardMenu
import com.kitakami.hibiki.elibrary.ui.bookshelf.components.ShelfTagDialog
import com.kitakami.hibiki.elibrary.ui.theme.BrandRed
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.launch
import java.io.File

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun BookShelfScreen(
    container: AppContainer,
    onOpenBook: (Long) -> Unit,
    onOpenSettings: () -> Unit,
    onOpenTags: () -> Unit,
) {
    val viewModel: BookShelfViewModel = viewModel {
        val app = this[ViewModelProvider.AndroidViewModelFactory.APPLICATION_KEY] as KHApplication
        BookShelfViewModel(app.container)
    }
    val baseUrl by container.baseUrl.collectAsStateWithLifecycle()
    val gridState = rememberLazyGridState()
    val snackbarHostState = remember { SnackbarHostState() }
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val drawerState = rememberDrawerState(DrawerValue.Closed)

    // Long-press card menu anchor id.
    var menuBookId by remember { mutableStateOf<Long?>(null) }
    // Dialogs.
    var shelfDialogBook by remember { mutableStateOf<Book?>(null) }
    var newShelfDialog by remember { mutableStateOf(false) }
    var bookToDelete by remember { mutableStateOf<Book?>(null) }
    var batchDeleteConfirm by remember { mutableStateOf(false) }

    // Infinite scroll.
    LaunchedEffect(gridState) {
        snapshotFlow { gridState.layoutInfo.visibleItemsInfo.lastOrNull()?.index ?: -1 }
            .distinctUntilChanged()
            .collect { index -> viewModel.loadMoreIfNeeded(index) }
    }

    // Surface transient messages.
    LaunchedEffect(viewModel.message) {
        val msg = viewModel.message ?: return@LaunchedEffect
        val text = context.getString(msg.resId, *msg.args.toTypedArray())
        snackbarHostState.showSnackbar(text)
        viewModel.consumeMessage()
    }

    val pickerLauncher = rememberLauncherForActivityResult(
        ActivityResultContracts.OpenDocument(),
    ) { uri: Uri? ->
        if (uri != null) {
            val file = copyToTempFile(context, uri)?.takeIf { it.isEpubOrPdf() }
            if (file == null) {
                scope.launch { snackbarHostState.showSnackbar(context.getString(R.string.upload_unsupported)) }
            } else {
                viewModel.upload(file)
            }
        }
    }

    ModalNavigationDrawer(
        drawerState = drawerState,
        drawerContent = {
            ModalDrawerSheet {
                DrawerContent(
                    viewModel = viewModel,
                    onSelectAll = { viewModel.selectTag("") },
                    onSelectNotInShelf = { viewModel.selectTag(NO_TAG_FILTER) },
                    onSelectTag = { viewModel.selectTag(it) },
                    onNewShelf = { newShelfDialog = true },
                    onOpenTags = onOpenTags,
                )
            }
        },
    ) {
        Scaffold(
            topBar = {
                ShelfTopBar(
                    selectionMode = viewModel.selectionMode,
                    selectedCount = viewModel.selectedIds.size,
                    onOpenDrawer = { scope.launch { drawerState.open() } },
                    onBackToNormal = { viewModel.updateSelectionMode(false) },
                    onDeleteSelected = { batchDeleteConfirm = true },
                    onAddBook = { pickerLauncher.launch(arrayOf("*/*")) },
                    onOpenSettings = onOpenSettings,
                )
            },
            snackbarHost = { SnackbarHost(snackbarHostState) },
        ) { pad ->
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(pad),
            ) {
                if (!viewModel.selectionMode) {
                    OutlinedTextField(
                        value = viewModel.keyword,
                        onValueChange = viewModel::onKeywordChange,
                        placeholder = { Text(stringResource(R.string.bookshelf_search)) },
                        singleLine = true,
                        leadingIcon = { Icon(Icons.Default.Search, contentDescription = null) },
                        trailingIcon = {
                            if (viewModel.keyword.isNotEmpty()) {
                                IconButton(onClick = { viewModel.onKeywordChange("") }) {
                                    Icon(Icons.Default.Close, contentDescription = null)
                                }
                            }
                        },
                        shape = RoundedCornerShape(12.dp),
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(horizontal = 16.dp, vertical = 8.dp),
                    )

                    FilterBar(
                        viewModel = viewModel,
                        modifier = Modifier.padding(bottom = 4.dp),
                    )
                }

                when {
                    viewModel.loading && viewModel.books.isEmpty() -> {
                        Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                            CircularProgressIndicator()
                        }
                    }

                    viewModel.error != null && viewModel.books.isEmpty() -> {
                        Column(
                            modifier = Modifier.fillMaxSize(),
                            verticalArrangement = Arrangement.Center,
                            horizontalAlignment = Alignment.CenterHorizontally,
                        ) {
                            Text(
                                text = stringResource(R.string.bookshelf_load_failed, viewModel.error ?: ""),
                                color = MaterialTheme.colorScheme.error,
                            )
                            Button(onClick = viewModel::retry, modifier = Modifier.padding(top = 12.dp)) {
                                Text(stringResource(R.string.bookshelf_retry))
                            }
                        }
                    }

                    viewModel.books.isEmpty() -> {
                        Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                            Text(
                                text = stringResource(R.string.bookshelf_empty),
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                                modifier = Modifier.padding(24.dp),
                            )
                        }
                    }

                    else -> {
                        LazyVerticalGrid(
                            columns = GridCells.Adaptive(minSize = 140.dp),
                            state = gridState,
                            contentPadding = PaddingValues(16.dp),
                            horizontalArrangement = Arrangement.spacedBy(16.dp),
                            verticalArrangement = Arrangement.spacedBy(20.dp),
                            modifier = Modifier.fillMaxSize(),
                        ) {
                            gridItems(viewModel.books, key = { it.id }) { book ->
                                Box {
                                    BookCard(
                                        book = book,
                                        coverUrl = coverUrlFor(baseUrl, book.id),
                                        onClick = { onOpenBook(book.id) },
                                        onLongClick = { menuBookId = book.id },
                                        selectionMode = viewModel.selectionMode,
                                        selected = book.id in viewModel.selectedIds,
                                        onToggleSelection = { viewModel.toggleSelection(book.id) },
                                    )
                                    BookCardMenu(
                                        book = book,
                                        expanded = menuBookId == book.id,
                                        onDismiss = { menuBookId = null },
                                        onRead = { onOpenBook(book.id) },
                                        onInfo = { onOpenBook(book.id) },
                                        onSetStatus = { viewModel.setReadStatus(book.id, it) },
                                        onAddToShelf = { shelfDialogBook = book },
                                        onReprocess = { viewModel.reprocess(book.id) },
                                        onExport = { onOpenBook(book.id) },
                                        onDelete = { bookToDelete = book },
                                    )
                                }
                            }
                            if (viewModel.loadingMore) {
                                item(span = { GridItemSpan(maxLineSpan) }) {
                                    Box(
                                        modifier = Modifier
                                            .fillMaxWidth()
                                            .padding(16.dp),
                                        contentAlignment = Alignment.Center,
                                    ) {
                                        CircularProgressIndicator(Modifier.size(28.dp), strokeWidth = 2.dp)
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    // ---- dialogs ------------------------------------------------------------

    shelfDialogBook?.let { book ->
        ShelfTagDialog(
            book = book,
            tags = viewModel.tags,
            onDismiss = { shelfDialogBook = null },
            onSave = { addIds, removeIds ->
                viewModel.updateBookTags(book.id, addIds, removeIds)
                shelfDialogBook = null
            },
        )
    }

    if (newShelfDialog) {
        NewShelfDialog(
            onDismiss = { newShelfDialog = false },
            onCreate = { name ->
                viewModel.createShelf(name)
                newShelfDialog = false
            },
        )
    }

    bookToDelete?.let { book ->
        AlertDialog(
            onDismissRequest = { bookToDelete = null },
            title = { Text(stringResource(R.string.menu_delete)) },
            text = { Text(stringResource(R.string.menu_delete_confirm, book.title)) },
            confirmButton = {
                TextButton(onClick = {
                    viewModel.deleteBook(book.id)
                    bookToDelete = null
                }) {
                    Text(stringResource(R.string.common_delete), color = BrandRed)
                }
            },
            dismissButton = {
                TextButton(onClick = { bookToDelete = null }) {
                    Text(stringResource(R.string.common_cancel))
                }
            },
        )
    }

    if (batchDeleteConfirm) {
        AlertDialog(
            onDismissRequest = { batchDeleteConfirm = false },
            title = { Text(stringResource(R.string.bookshelf_batch_delete)) },
            text = { Text(stringResource(R.string.bookshelf_batch_delete_confirm, viewModel.selectedIds.size)) },
            confirmButton = {
                TextButton(onClick = {
                    viewModel.deleteSelected()
                    batchDeleteConfirm = false
                }) {
                    Text(stringResource(R.string.common_delete), color = BrandRed)
                }
            },
            dismissButton = {
                TextButton(onClick = { batchDeleteConfirm = false }) {
                    Text(stringResource(R.string.common_cancel))
                }
            },
        )
    }

    // Upload progress dialog.
    when (val up = viewModel.uploadState) {
        is UploadState.Uploading -> {
            AlertDialog(
                onDismissRequest = {},
                title = { Text(stringResource(R.string.bookshelf_add_book)) },
                text = {
                    Column {
                        LinearProgressIndicator(
                            progress = { up.percent / 100f },
                            modifier = Modifier.fillMaxWidth(),
                        )
                        Spacer(Modifier.height(8.dp))
                        Text(
                            stringResource(
                                if (up.percent == 100) R.string.upload_reading_file else R.string.upload_chunking,
                                up.percent,
                            ),
                        )
                    }
                },
                confirmButton = {},
            )
        }
        is UploadState.Failed -> {
            AlertDialog(
                onDismissRequest = { viewModel.dismissUploadError() },
                title = { Text(stringResource(R.string.bookshelf_add_book)) },
                text = { Text(up.message) },
                confirmButton = {
                    TextButton(onClick = { viewModel.dismissUploadError() }) {
                        Text(stringResource(R.string.common_confirm))
                    }
                },
            )
        }
        UploadState.Idle -> {}
    }
}

// ---- helpers ---------------------------------------------------------------

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun ShelfTopBar(
    selectionMode: Boolean,
    selectedCount: Int,
    onOpenDrawer: () -> Unit,
    onBackToNormal: () -> Unit,
    onDeleteSelected: () -> Unit,
    onAddBook: () -> Unit,
    onOpenSettings: () -> Unit,
) {
    TopAppBar(
        title = {
            Text(
                if (selectionMode) stringResource(R.string.bookshelf_selected_count, selectedCount)
                else stringResource(R.string.bookshelf_title),
            )
        },
        navigationIcon = {
            if (selectionMode) {
                IconButton(onClick = onBackToNormal) {
                    Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.reader_back))
                }
            } else {
                IconButton(onClick = onOpenDrawer) {
                    Icon(Icons.Default.Menu, contentDescription = stringResource(R.string.bookshelf_title))
                }
            }
        },
        actions = {
            if (selectionMode) {
                IconButton(onClick = onDeleteSelected) {
                    Icon(Icons.Default.Delete, contentDescription = stringResource(R.string.bookshelf_batch_delete))
                }
            } else {
                IconButton(onClick = onAddBook) {
                    Icon(Icons.Default.Add, contentDescription = stringResource(R.string.bookshelf_add_book))
                }
                IconButton(onClick = onOpenSettings) {
                    Icon(Icons.Default.Settings, contentDescription = stringResource(R.string.bookshelf_settings))
                }
            }
        },
    )
}

@Composable
private fun FilterBar(
    viewModel: BookShelfViewModel,
    modifier: Modifier = Modifier,
) {
    var sortExpanded by remember { mutableStateOf(false) }
    var statusExpanded by remember { mutableStateOf(false) }

    Row(
        modifier = modifier
            .fillMaxWidth()
            .horizontalScroll(rememberScrollState())
            .padding(horizontal = 16.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        // Sort dropdown.
        Box {
            FilterChip(
                selected = false,
                onClick = { sortExpanded = true },
                label = { Text(stringResource(R.string.bookshelf_sort)) },
                leadingIcon = { Icon(Icons.Default.Sort, contentDescription = null, modifier = Modifier.size(16.dp)) },
            )
            DropdownMenu(expanded = sortExpanded, onDismissRequest = { sortExpanded = false }) {
                val fields = listOf(
                    "updated_at" to R.string.bookshelf_sort_updated_at,
                    "title" to R.string.bookshelf_sort_title,
                    "author" to R.string.bookshelf_sort_author,
                    "created_at" to R.string.bookshelf_sort_created_at,
                )
                fields.forEach { (field, labelRes) ->
                    DropdownMenuItem(
                        text = { Text(stringResource(labelRes)) },
                        trailingIcon = {
                            if (viewModel.sortField == field) Icon(Icons.Default.Check, null)
                        },
                        onClick = {
                            val order = if (field == viewModel.sortField && viewModel.sortOrder == "asc") "desc" else "asc"
                            viewModel.setSort(field, order)
                            sortExpanded = false
                        },
                    )
                }
                HorizontalDivider()
                DropdownMenuItem(
                    text = {
                        Text(
                            stringResource(
                                if (viewModel.sortOrder == "asc") R.string.bookshelf_sort_asc else R.string.bookshelf_sort_desc,
                            ),
                        )
                    },
                    onClick = {
                        viewModel.setSort(viewModel.sortField, if (viewModel.sortOrder == "asc") "desc" else "asc")
                        sortExpanded = false
                    },
                )
            }
        }

        // Status filter dropdown.
        Box {
            FilterChip(
                selected = viewModel.bookStatus.isNotEmpty(),
                onClick = { statusExpanded = true },
                label = {
                    Text(
                        when (viewModel.bookStatus) {
                            "reading" -> stringResource(R.string.status_reading)
                            "finished" -> stringResource(R.string.status_finished)
                            "unread" -> stringResource(R.string.status_unread)
                            else -> stringResource(R.string.bookshelf_filter_all)
                        },
                    )
                },
            )
            DropdownMenu(expanded = statusExpanded, onDismissRequest = { statusExpanded = false }) {
                listOf(
                    "" to R.string.bookshelf_filter_all,
                    "reading" to R.string.status_reading,
                    "finished" to R.string.status_finished,
                    "unread" to R.string.status_unread,
                ).forEach { (value, labelRes) ->
                    DropdownMenuItem(
                        text = { Text(stringResource(labelRes)) },
                        onClick = {
                            viewModel.filterByStatus(value)
                            statusExpanded = false
                        },
                    )
                }
            }
        }

        // Active tag chip (read-only indicator when filtering by a shelf).
        if (viewModel.tag.isNotEmpty()) {
            FilterChip(
                selected = true,
                onClick = { viewModel.selectTag("") },
                label = {
                    Text(
                        if (viewModel.tag == NO_TAG_FILTER) stringResource(R.string.bookshelf_not_in_shelf)
                        else viewModel.tag,
                    )
                },
            )
        }
    }
}

@Composable
private fun DrawerContent(
    viewModel: BookShelfViewModel,
    onSelectAll: () -> Unit,
    onSelectNotInShelf: () -> Unit,
    onSelectTag: (String) -> Unit,
    onNewShelf: () -> Unit,
    onOpenTags: () -> Unit,
) {
    Column(modifier = Modifier.fillMaxWidth()) {
        Text(
            text = stringResource(R.string.bookshelf_title),
            style = MaterialTheme.typography.titleLarge,
            modifier = Modifier.padding(16.dp),
        )
        HorizontalDivider()

        DrawerItem(
            label = stringResource(R.string.bookshelf_all_books),
            selected = viewModel.tag.isEmpty(),
            onClick = onSelectAll,
        )
        DrawerItem(
            label = stringResource(R.string.bookshelf_not_in_shelf),
            selected = viewModel.tag == NO_TAG_FILTER,
            onClick = onSelectNotInShelf,
        )

        HorizontalDivider(Modifier.padding(vertical = 4.dp))
        Text(
            text = stringResource(R.string.tags_title),
            style = MaterialTheme.typography.labelLarge,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp),
        )
        LazyColumn(Modifier.weight(1f, fill = false)) {
            lazyItems(viewModel.tags, key = { it.id }) { tag ->
                DrawerItem(
                    label = "${tag.name} (${tag.count})",
                    selected = viewModel.tag == tag.name,
                    onClick = { onSelectTag(tag.name) },
                )
            }
        }
        Row(modifier = Modifier.fillMaxWidth()) {
            TextButton(onClick = onNewShelf, modifier = Modifier.weight(1f)) {
                Icon(Icons.Default.Add, null)
                Spacer(Modifier.width(4.dp))
                Text(stringResource(R.string.bookshelf_new_shelf))
            }
            TextButton(onClick = onOpenTags, modifier = Modifier.weight(1f)) {
                Icon(Icons.Default.Menu, null)
                Spacer(Modifier.width(4.dp))
                Text(stringResource(R.string.tags_title))
            }
        }
    }
}

@Composable
private fun DrawerItem(label: String, selected: Boolean, onClick: () -> Unit) {
    Text(
        text = label,
        style = MaterialTheme.typography.bodyLarge,
        color = if (selected) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurface,
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .background(
                if (selected) MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.4f)
                else Color.Transparent,
            )
            .padding(horizontal = 16.dp, vertical = 12.dp),
    )
}

@Composable
private fun NewShelfDialog(
    onDismiss: () -> Unit,
    onCreate: (String) -> Unit,
) {
    var name by remember { mutableStateOf("") }
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(stringResource(R.string.bookshelf_new_shelf)) },
        text = {
            OutlinedTextField(
                value = name,
                onValueChange = { name = it },
                placeholder = { Text(stringResource(R.string.bookshelf_shelf_name)) },
                singleLine = true,
            )
        },
        confirmButton = {
            TextButton(
                onClick = { if (name.isNotBlank()) onCreate(name.trim()) },
                enabled = name.isNotBlank(),
            ) {
                Text(stringResource(R.string.common_confirm))
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) {
                Text(stringResource(R.string.common_cancel))
            }
        },
    )
}

/** Copies a content:// Uri into a cache temp file, returning null on failure. */
internal fun copyToTempFile(context: Context, uri: Uri): File? {
    return try {
        val displayName = context.contentResolver.query(
            uri, arrayOf(OpenableColumns.DISPLAY_NAME), null, null, null,
        )?.use { cursor ->
            if (cursor.moveToFirst()) {
                val idx = cursor.getColumnIndex(OpenableColumns.DISPLAY_NAME)
                if (idx >= 0) cursor.getString(idx) else null
            } else null
        } ?: "book"
        val safeName = displayName.ifBlank { "book" }.replace(Regex("[\\\\/:*?\"<>|]"), "_")
        val target = File(context.cacheDir, "upload_${System.currentTimeMillis()}_$safeName")
        context.contentResolver.openInputStream(uri)?.use { input ->
            target.outputStream().use { output -> input.copyTo(output) }
        }
        target
    } catch (e: Exception) {
        null
    }
}

private fun File.isEpubOrPdf(): Boolean {
    val ext = extension.lowercase()
    return ext == "epub" || ext == "pdf"
}

internal fun coverUrlFor(baseUrl: String, bookId: Long): String {
    val base = baseUrl.trim().trimEnd('/')
    return "$base/api/v1/books/cover?id=$bookId"
}
