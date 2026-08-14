package com.kitakami.hibiki.elibrary.ui.bookshelf.components

import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.res.stringResource
import com.kitakami.hibiki.elibrary.R
import com.kitakami.hibiki.elibrary.data.model.Book

/**
 * Long-press action menu shown for a book card. Mirrors the Web client's
 * `BookCardMenu.vue`: read / info / status toggle / add to shelf / reprocess /
 * export / delete.
 */
@Composable
fun BookCardMenu(
    book: Book,
    expanded: Boolean,
    onDismiss: () -> Unit,
    onRead: () -> Unit,
    onInfo: () -> Unit,
    onSetStatus: (String) -> Unit,
    onAddToShelf: () -> Unit,
    onReprocess: () -> Unit,
    onExport: () -> Unit,
    onDelete: () -> Unit,
) {
    DropdownMenu(expanded = expanded, onDismissRequest = onDismiss) {
        DropdownMenuItem(
            text = { Text(stringResource(R.string.menu_read)) },
            onClick = { onDismiss(); onRead() },
        )
        DropdownMenuItem(
            text = { Text(stringResource(R.string.menu_info)) },
            onClick = { onDismiss(); onInfo() },
        )
        if (book.read_status != "reading") {
            DropdownMenuItem(
                text = { Text(stringResource(R.string.menu_mark_reading)) },
                onClick = { onDismiss(); onSetStatus("reading") },
            )
        }
        if (book.read_status != "finished") {
            DropdownMenuItem(
                text = { Text(stringResource(R.string.menu_mark_finished)) },
                onClick = { onDismiss(); onSetStatus("finished") },
            )
        }
        if (book.read_status != "unread") {
            DropdownMenuItem(
                text = { Text(stringResource(R.string.menu_mark_unread)) },
                onClick = { onDismiss(); onSetStatus("unread") },
            )
        }
        DropdownMenuItem(
            text = { Text(stringResource(R.string.menu_add_to_shelf)) },
            onClick = { onDismiss(); onAddToShelf() },
        )
        DropdownMenuItem(
            text = { Text(stringResource(R.string.menu_reprocess)) },
            onClick = { onDismiss(); onReprocess() },
        )
        DropdownMenuItem(
            text = { Text(stringResource(R.string.menu_export)) },
            onClick = { onDismiss(); onExport() },
        )
        DropdownMenuItem(
            text = { Text(stringResource(R.string.menu_delete), color = androidx.compose.ui.graphics.Color(0xFFE74C3C)) },
            onClick = { onDismiss(); onDelete() },
        )
    }
}
