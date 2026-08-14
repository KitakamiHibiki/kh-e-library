package com.kitakami.hibiki.elibrary.ui.reader

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.EmojiEvents
import androidx.compose.material3.Button
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import com.kitakami.hibiki.elibrary.R
import com.kitakami.hibiki.elibrary.ui.theme.BrandGreen

/** Full-screen "book finished" page (design doc §6.4/§6.5). */
@Composable
fun ReadCompleteOverlay(
    onMarkFinished: () -> Unit,
    onReadAgain: () -> Unit,
    onBackToLast: () -> Unit,
) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(MaterialTheme.colorScheme.background),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Icon(
            imageVector = Icons.Default.EmojiEvents,
            contentDescription = null,
            tint = BrandGreen,
            modifier = Modifier.padding(bottom = 16.dp),
        )
        Text(
            text = stringResource(R.string.reader_complete_title),
            style = MaterialTheme.typography.headlineMedium,
        )
        Spacer(Modifier.height(8.dp))
        Text(
            text = stringResource(R.string.reader_complete_message),
            style = MaterialTheme.typography.bodyLarge,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Spacer(Modifier.height(32.dp))
        Button(onClick = onMarkFinished, modifier = Modifier.fillMaxWidth().padding(horizontal = 40.dp)) {
            Text(stringResource(R.string.reader_mark_finished))
        }
        Spacer(Modifier.height(12.dp))
        OutlinedButton(onClick = onReadAgain, modifier = Modifier.fillMaxWidth().padding(horizontal = 40.dp)) {
            Text(stringResource(R.string.reader_read_again))
        }
        Spacer(Modifier.height(12.dp))
        OutlinedButton(onClick = onBackToLast, modifier = Modifier.fillMaxWidth().padding(horizontal = 40.dp)) {
            Text(stringResource(R.string.reader_back_to_last))
        }
    }
}
