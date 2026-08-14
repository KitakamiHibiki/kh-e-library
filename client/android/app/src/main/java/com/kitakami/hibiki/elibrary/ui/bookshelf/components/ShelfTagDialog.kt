package com.kitakami.hibiki.elibrary.ui.bookshelf.components

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Checkbox
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.kitakami.hibiki.elibrary.R
import com.kitakami.hibiki.elibrary.data.model.Book
import com.kitakami.hibiki.elibrary.data.model.Tag

/**
 * "Add to shelf" dialog: multi-selects tags for a book. Diff vs the book's
 * current tags is computed on save and passed out as add/remove id lists.
 */
@Composable
fun ShelfTagDialog(
    book: Book,
    tags: List<Tag>,
    onDismiss: () -> Unit,
    onSave: (addIds: List<Long>, removeIds: List<Long>) -> Unit,
) {
    val currentNames = book.tags.toSet()
    var selectedIds by remember(book, tags) {
        mutableStateOf(tags.filter { it.name in currentNames }.map { it.id }.toSet())
    }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(stringResource(R.string.menu_add_to_shelf)) },
        text = {
            if (tags.isEmpty()) {
                Text(stringResource(R.string.tags_select_hint))
            } else {
                LazyColumn(Modifier.height(300.dp)) {
                    items(tags, key = { it.id }) { tag ->
                        Row(
                            verticalAlignment = Alignment.CenterVertically,
                            modifier = Modifier
                                .fillMaxWidth()
                                .clickable {
                                    selectedIds = if (tag.id in selectedIds) selectedIds - tag.id else selectedIds + tag.id
                                }
                                .padding(vertical = 6.dp),
                        ) {
                            Checkbox(
                                checked = tag.id in selectedIds,
                                onCheckedChange = {
                                    selectedIds = if (tag.id in selectedIds) selectedIds - tag.id else selectedIds + tag.id
                                },
                            )
                            Text(tag.name, modifier = Modifier.padding(start = 8.dp))
                        }
                    }
                }
            }
        },
        confirmButton = {
            TextButton(onClick = {
                val initialIds = tags.filter { it.name in currentNames }.map { it.id }.toSet()
                val addIds = selectedIds.filter { it !in initialIds }
                val removeIds = initialIds.filter { it !in selectedIds }
                onSave(addIds, removeIds)
            }) {
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
