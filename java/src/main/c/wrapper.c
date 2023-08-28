#include <jni.h>
#include <stdlib.h>
#include "../../../target/native/librulestone-go.h"

#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wincompatible-pointer-types-discards-qualifiers"

JNIEXPORT jint JNICALL Java_com_atlasgurus_rulestone_Rulestone_NewRuleEngine(JNIEnv *env, jclass cls, jstring jstr) {
    const char *rules_path = (*env)->GetStringUTFChars(env, jstr, 0);
    int rule_engine_id = NewRuleEngine(rules_path);
    (*env)->ReleaseStringUTFChars(env, jstr, rules_path);
    return rule_engine_id;
}

JNIEXPORT jintArray JNICALL Java_com_atlasgurus_rulestone_Rulestone_Match(JNIEnv *env, jclass cls, jint rule_engine_id, jstring jstr) {
    const char *input = (*env)->GetStringUTFChars(env, jstr, 0);
    Matches* matches = Match(rule_engine_id, input);
    jintArray result = (*env)->NewIntArray(env, matches->len);
    (*env)->SetIntArrayRegion(env, result, 0, matches->len, matches->matches);
    free(matches->matches);
    free(matches);
    (*env)->ReleaseStringUTFChars(env, jstr, input);
    return result;
}

JNIEXPORT jobject JNICALL Java_com_atlasgurus_rulestone_Rulestone_GetRuleMetadata(JNIEnv *env, jclass cls, jint rule_engine_id, jint rule_id) {
    Metadata* metadata = GetRuleMetadata(rule_engine_id, rule_id);

    if (metadata == NULL) {
        return NULL;
    }

    jclass ruleMetadataClass = (*env)->FindClass(env, "com/atlasgurus/rulestone/RuleMetadata");
    jmethodID constructor = (*env)->GetMethodID(env, ruleMetadataClass, "<init>", "(Ljava/util/Map;)V");
    jclass hashMapClass = (*env)->FindClass(env, "java/util/HashMap");
    jmethodID hashMapConstructor = (*env)->GetMethodID(env, hashMapClass, "<init>", "()V");
    jobject hashMap = (*env)->NewObject(env, hashMapClass, hashMapConstructor);
    jmethodID hashMapPut = (*env)->GetMethodID(env, hashMapClass, "put", "(Ljava/lang/Object;Ljava/lang/Object;)Ljava/lang/Object;");

    for (int i = 0; i < metadata->size; i++ ) {
        jstring key = (*env)->NewStringUTF(env, metadata->metadata[i*2]); //
        jstring value = (*env)->NewStringUTF(env, metadata->metadata[i*2+1]);
        (*env)->CallObjectMethod(env, hashMap, hashMapPut, key, value);
        free(metadata->metadata[i*2]);
        free(metadata->metadata[i*2+1]);
    }

    free(metadata->metadata);

    return (*env)->NewObject(env, ruleMetadataClass, constructor, hashMap);
}