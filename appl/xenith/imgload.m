Imgload: module {
	PATH: con "/dis/xenith/imgload.dis";

	# Progress callback info for progressive decoding
	ImgProgress: adt {
		image: ref Draw->Image;  # Image being decoded
		rowsdone: int;           # Rows decoded so far
		rowstotal: int;          # Total rows
	};

	init: fn(d: ref Draw->Display);
	readimage: fn(path: string): (ref Draw->Image, string);
	readimagedata: fn(data: array of byte, hint: string): (ref Draw->Image, string);

	# Progressive image decode - sends progress updates to channel
	# Returns (image, error) when complete
	readimagedataprogressive: fn(data: array of byte, hint: string,
	                             progress: chan of ref ImgProgress): (ref Draw->Image, string);
};
