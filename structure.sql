CREATE TABLE `nameservers` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `ip` varchar(255) NOT NULL,
  `name` varchar(255) DEFAULT NULL,
  `state` varchar(255) NOT NULL DEFAULT 'new',
  `state_changed_at` datetime DEFAULT NULL,
  `error` varchar(255) DEFAULT NULL,
  `country_id` char(2) DEFAULT NULL,
  `city` varchar(255) DEFAULT NULL,
  `checked_at` datetime DEFAULT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `version` varchar(255) DEFAULT NULL,
  `dnssec` tinyint(1) DEFAULT NULL,
  `reliability` float(3,2) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `index_nameservers_on_ip` (`ip`),
  KEY `index_nameservers_on_state` (`state`),
  KEY `country_state_checked` (`country_id`,`state`,`checked_at`),
  KEY `index_nameservers_on_version` (`version`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;


CREATE TABLE `nameserver_checks` (
  `nameserver_id` int(11) NOT NULL,
  `result` tinyint(1) NOT NULL,
  `error` varchar(255) DEFAULT NULL,
  `created_at` datetime NOT NULL,
  KEY `index_nameserver_checks_on_nameserver_id_and_result` (`nameserver_id`,`result`),
  CONSTRAINT `nameserver_checks_ibfk_1` FOREIGN KEY (`nameserver_id`) REFERENCES `nameservers` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8;


DELIMITER ;;
CREATE TRIGGER update_checks BEFORE UPDATE ON nameservers
    FOR EACH ROW
    BEGIN
    IF NEW.error = "" THEN
        SET NEW.error = NULL;
    END IF;

    IF NEW.checked_at != OLD.checked_at THEN
        INSERT INTO nameserver_checks (nameserver_id, result, error, created_at) VALUES (NEW.id, NEW.error IS null, error, NEW.checked_at);
        SET NEW.reliability = (SELECT SUM(result)/COUNT(*) FROM nameserver_checks WHERE nameserver_id=NEW.id);
    END IF;

    IF NEW.state != OLD.state THEN
        SET NEW.state_changed_at = NOW();
    END IF;
END;;
DELIMITER ;
