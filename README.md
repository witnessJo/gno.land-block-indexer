author: witnessjo
date: \[2025-08-13 Wed\]
title: Gno.land Block Indexer

# Overview

Gno.land Block Indexer는 Gno 블록체인의 블록, 트랜잭션, 전송, 계정
데이터를 실시간으로 수집하고 인덱싱하는 시스템입니다.

# Architecture

이 프로젝트는 다음 세 가지 주요 컴포넌트로 구성됩니다:

## Components

### Block Synchronizer (`cmd/block-synchronizer`)

-   Gno 블록체인으로부터 새로운 블록을 실시간으로 동기화
-   메시지 브로커를 통해 이벤트 발행

### Event Processor (`cmd/event-processor`)

-   메시지 브로커로부터 이벤트를 수신
-   트랜잭션과 전송 데이터를 처리 및 파싱
-   블록 데이터를 데이터베이스에 저장
-   계정 정보 업데이트

### Indexer REST API (`cmd/indexer-rest`)

-   인덱싱된 데이터에 대한 REST API 제공
-   GraphQL 엔드포인트 지원
-   로컬 캐싱을 통한 성능 최적화

# Technology Stack

-   **Language**: Go 1.23+
-   **Database**: PostgreSQL
-   **ORM**: Ent (Facebook's Entity Framework for Go)
-   **Message Broker**: AWS SNS/SQS (LocalStack 지원)
-   **Web Framework**: Gin
-   **API**: REST
-   **Containerization**: Docker Compose / Kubernetes

# Getting Started

## Prerequisites

-   Go 1.23 이상
-   PostgreSQL
-   Docker & Docker Compose

## Installation

1.  Repository 클론:

    ``` shell
    git clone <repository-url>
    cd gno.land-block-indexer
    ```

2.  의존성 설치:

    ``` shell
    go mod download
    ```

3.  Ent 설치 및 코드 생성:

    ``` shell
    make ent-install
    make ent
    ```

## Running the Application

### Using Make

1.  모든 바이너리 빌드:

    ``` shell
    make all
    ```

2.  인프라 시작 (PostgreSQL, LocalStack):

    ``` shell
    make infra
    ```

3.  각 컴포넌트 실행:

    ``` shell
    ./bin/block-synchronizer
    ./bin/event-processor
    ./bin/indexer-rest
    ```

### Using Docker Compose

``` shell
docker-compose -f containers-compose.yaml up
```

### Using Kubernetes

``` shell
kubectl apply -f containers-kube.yaml
```

# Database Schema

데이터베이스 스키마는 Ent를 사용하여 정의되며, 다음 엔티티들로
구성됩니다:

-   **Block**: 블록체인 블록 정보
-   **Transaction**: 트랜잭션 데이터
-   **Transfer**: 토큰 전송 정보
-   **Account**: 계정 정보
-   **RestoreHistory**: 복원 히스토리

스키마 정의는 `ent/schema/` 디렉토리 또는 /schema.sql 파일에서 확인할 수
있습니다.

# API Documentation

REST API 엔드포인트는 Postman Collection으로 문서화되어 있습니다:

-   `gno.block-indexer.postman_collection.json`

# Development

## Building Individual Components

``` shell
# Block Synchronizer
make bs-bin

# Event Processor
make ep-bin

# REST API Server
make rest-bin
```

## Code Generation

Ent 스키마 변경 후 코드 재생성:

``` shell
make ent
```

## Testing

``` shell
go test ./...
```

# <span class="todo TODO">TODO</span>  [section]

-   모듈의 controller.go 내의 설정 .env로 변경
