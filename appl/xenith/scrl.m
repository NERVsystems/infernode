Scroll : module {
	PATH : con "/dis/xenith/scrl.dis";

	init : fn(mods : ref Dat->Mods);
	scrsleep : fn(n : int);
	scrdraw : fn(t : ref Textm->Text);
	scrresize : fn();
	scroll : fn(t : ref Textm->Text, but : int);

	# Non-blocking scroll API (Phase 5b)
	scrollstart : fn(t : ref Textm->Text, but : int);
	scrollupdate : fn();
	scrollend : fn();
};