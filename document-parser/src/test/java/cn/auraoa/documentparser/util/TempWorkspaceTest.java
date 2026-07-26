package cn.auraoa.documentparser.util;

import org.junit.jupiter.api.Test;

import java.nio.file.Files;
import java.nio.file.Path;

import static org.assertj.core.api.Assertions.assertThat;

class TempWorkspaceTest {

    @Test
    void closesAndDeletesNestedFiles() throws Exception {
        Path directory;
        try (TempWorkspace workspace = TempWorkspace.create()) {
            directory = workspace.directory();
            Path nested = workspace.resolve("nested");
            Files.createDirectories(nested);
            Files.writeString(nested.resolve("temporary.txt"), "temporary");
            assertThat(Files.exists(directory)).isTrue();
        }

        assertThat(Files.exists(directory)).isFalse();
    }
}
