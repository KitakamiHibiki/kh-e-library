package com.kitakami.hibiki.elibrary.ui.bookshelf.components

import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.background
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Checkbox
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import coil.compose.AsyncImage
import coil.request.ImageRequest
import com.kitakami.hibiki.elibrary.R
import com.kitakami.hibiki.elibrary.data.model.Book
import com.kitakami.hibiki.elibrary.ui.theme.BrandBlue
import com.kitakami.hibiki.elibrary.ui.theme.BrandGreen
import com.kitakami.hibiki.elibrary.ui.theme.BrandRed

@OptIn(ExperimentalFoundationApi::class)
@Composable
fun BookCard(
    book: Book,
    coverUrl: String,
    onClick: () -> Unit,
    onLongClick: () -> Unit,
    selectionMode: Boolean = false,
    selected: Boolean = false,
    onToggleSelection: () -> Unit,
) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .combinedClickable(
                onClick = { if (selectionMode) onToggleSelection() else onClick() },
                onLongClick = { if (!selectionMode) onLongClick() },
            ),
    ) {
        CoverBox(
            book = book,
            coverUrl = coverUrl,
            selectionMode = selectionMode,
            selected = selected,
            onToggleSelection = onToggleSelection,
        )
        Spacer(Modifier.height(8.dp))
        Text(
            text = book.title,
            style = MaterialTheme.typography.titleSmall,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.fillMaxWidth(),
        )
        Text(
            text = book.author.ifBlank { stringResource(R.string.book_unknown_author) },
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.fillMaxWidth(),
        )
    }
}

@Composable
private fun CoverBox(
    book: Book,
    coverUrl: String,
    selectionMode: Boolean,
    selected: Boolean,
    onToggleSelection: () -> Unit,
) {
    val coverShape = RoundedCornerShape(8.dp)
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .aspectRatio(3f / 4f)
            .clip(coverShape)
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

        // Reading status badge on the cover (only when reading/finished).
        if (!selectionMode && (book.read_status == "reading" || book.read_status == "finished")) {
            StatusBadge(
                status = book.read_status,
                modifier = Modifier
                    .align(Alignment.TopStart)
                    .padding(6.dp),
            )
        }

        // Processing / failed overlay.
        if (book.isProcessing || book.isFailed) {
            Box(
                modifier = Modifier
                    .matchParentSize()
                    .background(Color.Black.copy(alpha = 0.5f)),
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    text = stringResource(if (book.isProcessing) R.string.bookshelf_processing else R.string.bookshelf_failed),
                    color = Color.White,
                    style = MaterialTheme.typography.labelMedium,
                )
            }
        }

        // Selection checkbox (top-right in multi-select mode).
        if (selectionMode) {
            Checkbox(
                checked = selected,
                onCheckedChange = { onToggleSelection() },
                modifier = Modifier
                    .align(Alignment.TopEnd)
                    .padding(6.dp),
            )
        }
    }
}

@Composable
private fun StatusBadge(status: String, modifier: Modifier = Modifier) {
    val label = stringResource(
        when (status) {
            "reading" -> R.string.status_reading
            "finished" -> R.string.status_finished
            else -> R.string.status_unread
        },
    )
    val dotColor = when (status) {
        "reading" -> BrandBlue
        "finished" -> BrandGreen
        else -> Color.Gray
    }
    Surface(
        color = Color.Black.copy(alpha = 0.55f),
        shape = RoundedCornerShape(4.dp),
        modifier = modifier,
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Box(Modifier.size(6.dp).background(dotColor, CircleShape))
            Spacer(Modifier.width(4.dp))
            Text(text = label, color = Color.White, style = MaterialTheme.typography.labelSmall)
        }
    }
}
