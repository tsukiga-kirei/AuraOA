package cn.auraoa.documentparser.parser;

import java.nio.file.Path;

/**
 * 单一文档格式解析器。
 */
public interface DocumentFormatParser {

    String fileType();

    ParseResult parse(Path input);
}
