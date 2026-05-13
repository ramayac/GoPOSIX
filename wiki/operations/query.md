# Query Workflow

## Goal

Answer repo questions from the wiki first so the agent does not start from zero every time.

## Procedure

1. Read `wiki/index.md`.
2. Read the latest relevant entries in `wiki/log.md`.
3. Search the wiki for the topic.
4. Read only the linked pages needed to answer the question.
5. Read source files only when the wiki lacks enough evidence.
6. If the answer is durable, write it back into the wiki.

## Shell-First Search

```bash
wiki-engine search <keyword>
wiki-engine list
wiki-engine headings
```

## Durable Answer Rule

File the answer back into the wiki when it is any of these:

- A stable architecture explanation.
- A repo workflow that will be reused.
- A non-obvious cross-file connection.
- A limitation, exclusion, or decision that future sessions should not rediscover.
