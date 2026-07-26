package cn.auraoa.documentparser;

import cn.auraoa.documentparser.config.ParserProperties;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.context.properties.EnableConfigurationProperties;

@SpringBootApplication
@EnableConfigurationProperties(ParserProperties.class)
public class DocumentParserApplication {

    public static void main(String[] args) {
        SpringApplication.run(DocumentParserApplication.class, args);
    }
}
