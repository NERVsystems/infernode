#include <lib9.h>
#include <math.h>

int
isNaN(double d)
{
	return isnan(d);
}

int
isInf(double d, int sign)
{
	if(sign < 0)
		return isinf(d) < 0;
	return isinf(d);
}
