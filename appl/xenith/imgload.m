Imgload: module {
	PATH: con "/dis/xenith/imgload.dis";

	init: fn(d: ref Draw->Display);
	readimage: fn(path: string): (ref Draw->Image, string);
};
