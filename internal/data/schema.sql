DROP TABLE IF EXISTS users;
-- IDEA: ADD TRUSTED DEVICES HASH
CREATE TABLE users(
	id				INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
	username		VARCHAR(128) NOT NULL,
	email			VARCHAR(320) NOT NULL,
);

DROP TABLE IF EXISTS subscriptions;
CREATE TABLE IF NOT EXISTS subscriptions(
	id				INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
	user_id			INTEGER FOREIGN KEY AUTOINCREMENT NOT NULL,
	device_hash		VARCHAR(40), -- uuid will be 36 chars.
	expire_date		DATE
	-- add some admin options if you want.
);

-- can't help but we have to do this way to make more space efficient.
DROP TABLE IF EXISTS friends;
CREATE TABLE IF NOT EXISTS friends(
	-- didn't add primary key cause that's not scalable. the max number of connection is number of users^2. impossible to scale.
	request_maker_id		BIGINT UNSIGNED NOT NULL,
	request_approver_id		BIGINT UNSIGNED NOT NULL,
	friends_room_id					BIGINT UNSIGNED,	-- create a room first?
	status					TINYINT NOT NULL DEFAULT 0,	-- 0 means requested, 1 means accepted.
	mutual_friend_count		BIGINT UNSIGNED,	-- for caching if NULL then calculate.
	FOREIGN KEY(request_maker_id) REFERENCES users(id),
	FOREIGN KEY(request_approver_id) REFERENCES users(id),
	FOREIGN KEY(friends_room_id) REFERENCES rooms(room_id)
);

DROP TABLE IF EXISTS room_subscribers;
CREATE TABLE IF NOT EXISTS room_subscribers(
	sub_id	SERIAL,
	subscribers_room_id BIGINT UNSIGNED NOT NULL,
	user_id BIGINT UNSIGNED NOT NULL,
	PRIMARY KEY(sub_id),
	FOREIGN KEY(subscribers_room_id) REFERENCES rooms(room_id),
	FOREIGN KEY(user_id) REFERENCES users(id)
);

DROP TABLE IF EXISTS messages;
CREATE TABLE IF NOT EXISTS messages(
	message_id					SERIAL,
	sender_id			BIGINT UNSIGNED NOT NULL,
	messages_room_id				BIGINT UNSIGNED NOT NULL,
	message				TEXT NOT NULL,
	created_timestamp	TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY(sender_id) REFERENCES users(id),
	FOREIGN KEY(messages_room_id) REFERENCES rooms(room_id),
	PRIMARY KEY(message_id)
);

-- -- adding mock users
-- pw - theint
INSERT INTO users (username, password, email, id_hash) VALUES('theint', '2e8f4fcb20968c70cbe3bea3ad85cacfac31e8bc79b7963a0daf3010bc2a918f', 'theint@gmail.com', '3IkmTh9rujIC0kcnWmddTXWWedkidWXaE4xsqw4=');
INSERT INTO users (username, password, email, id_hash) VALUES('kgmyat', 'ee8d775e547e97c58567a87387ddcba96b42f1b271d6df77666a12ee459b23b7', 'kgmyat@gmail.com', 'XZ87MZLktOzGxACduU9n1y1xUldSWk1r_7ihonU=');
-- pw - htetmyat
INSERT INTO users (username, password, email, id_hash) VALUES('htetmyatthar', '78795a9e5c63857b39feec691e03cb1fd3c764819c6926e90f375dfa0fb64124', 'htetmyathm2002@gmail.com', 'gtVZYflTXZKreA2yj-uf7Dld_hx-TBYJ8c6Klxw=');

-- adding mock friends
-- first_user is the request maker
-- second_user is the accepting one
INSERT INTO friends(request_maker_id, request_approver_id, status, mutual_friend_count) VALUES('1', '2', '0', 10);
INSERT INTO friends(request_maker_id, request_approver_id, status, mutual_friend_count) VALUES('2', '3', '0', 10);
INSERT INTO friends(request_maker_id, request_approver_id, status, mutual_friend_count) VALUES('3', '1', '0', 10);
