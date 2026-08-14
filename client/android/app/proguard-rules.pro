# Keep kotlinx.serialization generated serializers
-keepattributes *Annotation*, InnerClasses
-dontnote kotlinx.serialization.AnnotationsKt
-keep,includedescriptorclasses class com.kitakami.hibiki.elibrary.**$$serializer { *; }
-keepclassmembers class com.kitakami.hibiki.elibrary.** {
    *** Companion;
}
-keepclasseswithmembers class com.kitakami.hibiki.elibrary.** {
    kotlinx.serialization.KSerializer serializer(...);
}
