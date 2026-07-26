# syntax=docker/dockerfile:1.7
FROM maven:3.9.11-eclipse-temurin-21 AS build
WORKDIR /workspace

COPY pom.xml ./
RUN --mount=type=cache,target=/root/.m2 \
    mvn -B -ntp dependency:go-offline

COPY src ./src
RUN --mount=type=cache,target=/root/.m2 \
    mvn -B -ntp clean package

FROM eclipse-temurin:21-jre-jammy

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
       curl \
       fontconfig \
       fonts-noto-cjk \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --system --gid 10001 auraoa \
    && useradd --system --uid 10001 --gid auraoa --home-dir /app --shell /usr/sbin/nologin auraoa \
    && mkdir -p /app /tmp/document-parser \
    && chown -R auraoa:auraoa /app /tmp/document-parser

WORKDIR /app
COPY --from=build --chown=auraoa:auraoa /workspace/target/document-parser-*.jar /app/app.jar

USER auraoa
EXPOSE 8090

ENV JAVA_TOOL_OPTIONS="-XX:MaxRAMPercentage=75.0 -Djava.awt.headless=true -Dfile.encoding=UTF-8 -Djava.io.tmpdir=/tmp/document-parser"

HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
  CMD curl --fail --silent --show-error "http://127.0.0.1:${SERVER_PORT:-8090}/health" >/dev/null || exit 1

ENTRYPOINT ["java", "-jar", "/app/app.jar"]
