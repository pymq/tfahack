CREATE TABLE IF NOT EXISTS "Recipients"
(
    "RecipientId"     INTEGER NOT NULL UNIQUE,
    "RecipientName"   TEXT    NOT NULL,
    "RecipientTGName" TEXT,
    "RecipientTGId"   TEXT,
    "RecipientRoom"   TEXT    NOT NULL,
    PRIMARY KEY ("RecipientId" AUTOINCREMENT)
);

CREATE TABLE IF NOT EXISTS "MailingList"
(
    "ListId"   INTEGER NOT NULL UNIQUE,
    "SenderId" INTEGER COLLATE BINARY,
    "ListName" TEXT,
    PRIMARY KEY ("ListId" AUTOINCREMENT)
);

CREATE TABLE IF NOT EXISTS "Messages"
(
    "MessageId"    INTEGER NOT NULL DEFAULT 1 UNIQUE,
    "SenderId"     INTEGER NOT NULL,
    "RecipientId"  INTEGER NOT NULL,
    "TopicId"      INTEGER NOT NULL,
    "PostId"       INTEGER NOT NULL,
    "Message"      TEXT    NOT NULL,
    "SendDateTime" INTEGER,
    "React"        TEXT,
    "Read"         INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY ("MessageId" AUTOINCREMENT)
);

CREATE TABLE IF NOT EXISTS "Posts"
(
    "PostId"   INTEGER NOT NULL UNIQUE,
    "SenderId" INTEGER NOT NULL,
    "ListId"   INTEGER NOT NULL,
    "TopicId"  INTEGER NOT NULL,
    "Message"  TEXT,
    PRIMARY KEY ("PostId" AUTOINCREMENT)
);

CREATE TABLE IF NOT EXISTS "Senders"
(
    "SenderId"     INTEGER NOT NULL UNIQUE,
    "SenderName"   TEXT    NOT NULL,
    "SenderTGName" TEXT    NOT NULL,
    "SenderTGId"   TEXT    NOT NULL,
    "SenderRoom"   TEXT    NOT NULL,
    PRIMARY KEY ("SenderId" AUTOINCREMENT)
);

CREATE TABLE IF NOT EXISTS "Topics"
(
    "TopicId"  INTEGER NOT NULL UNIQUE,
    "SenderId" INTEGER NOT NULL,
    "Topic"    TEXT,
    PRIMARY KEY ("TopicId" AUTOINCREMENT)
)