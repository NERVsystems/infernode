#include "os.h"
#include <mp.h>
#include <libsec.h>

EGpriv*
eggen(int nlen, int rounds)
{
	EGpub *pub;
	EGpriv *priv;

	priv = egprivalloc();
	pub = &priv->pub;
	pub->p = mpnew(0);
	pub->alpha = mpnew(0);
	pub->key = mpnew(0);
	priv->secret = mpnew(0);

	/* Use pre-computed RFC 3526 params if available for this size */
	if(!getdhparams(nlen, pub->p, pub->alpha))
		gensafeprime(pub->p, pub->alpha, nlen, rounds);

	mprand(nlen-1, genrandom, priv->secret);
	mpexp(pub->alpha, priv->secret, pub->p, pub->key);
	return priv;
}
