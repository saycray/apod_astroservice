CREATE TABLE public.PICTURES (
	"date"            DATE NOT NULL,
	"title"           VARCHAR(255) NOT NULL,
	"url"             VARCHAR(255),
	"hd_url"          VARCHAR(255),
	"thumbnail_url"   VARCHAR(255),
	"media_type"      VARCHAR(16) NOT NULL,
    "copyright"       VARCHAR(255),
	"explanation"     VARCHAR(2048) NOT NULL
);

CREATE UNIQUE INDEX pictures_date_idx ON public.pictures ("date");
