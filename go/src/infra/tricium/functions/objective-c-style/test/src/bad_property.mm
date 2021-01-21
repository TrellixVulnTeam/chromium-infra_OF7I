
@implementation Bad

@property id<Foo> foo;
@property Foo* foo;
@property(atomic) Foo* foo;
@property(atomic) id<Foo> foo;
@property(readonly) Foo* foo;
@property(nonatomic, readwrite) id<Foo> foo;

@end
